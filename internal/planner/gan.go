package planner

import (
	"sort"

	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/templates"
)

// GANPlanParams — параметры плана, декодированные из 19-мерного вектора GAN.
// Дополнительные измерения (3..18) задают подходы, повторения, отдых и выбор
// упражнений, поэтому GAN-вектор реально формирует план, а не только интенсивность.
type GANPlanParams struct {
	Intensity        float64
	DurationMinutes  int
	SessionsPerWeek  int
	Sets             int
	Reps             int
	RestSeconds      int
	ExerciseCount    int
	SelectionWeights []float64
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampIntValue(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// DecodeGANVector декодирует 19-мерный вектор GAN в параметры плана.
//
//	0  — интенсивность (0..1)
//	1  — длительность (0..1) → 30..90 мин
//	2  — частота (0..1) → 1..7 занятий/нед
//	4  — подходы (0..1) → 2..6
//	5  — повторения (0..1) → 6..21
//	6  — отдых (0..1) → 30..120 сек
//	7  — число упражнений (0..1) → 0..6 (0 = все из пула)
//	8+ — веса выбора упражнений из пула класса
func DecodeGANVector(vec []float64) GANPlanParams {
	params := GANPlanParams{
		Intensity:       0.5,
		DurationMinutes: 45,
		SessionsPerWeek: 3,
		Sets:            3,
		Reps:            12,
		RestSeconds:     60,
	}
	if len(vec) == 0 {
		return params
	}
	get := func(i int) (float64, bool) {
		if i < len(vec) {
			return clamp01(vec[i]), true
		}
		return 0, false
	}
	if v, ok := get(0); ok {
		params.Intensity = v
	}
	if v, ok := get(1); ok {
		params.DurationMinutes = 30 + int(v*60)
	}
	if v, ok := get(2); ok {
		params.SessionsPerWeek = clampIntValue(1+int(v*6), 1, 7)
	}
	if v, ok := get(4); ok {
		params.Sets = clampIntValue(2+int(v*4), 1, 8)
	}
	if v, ok := get(5); ok {
		params.Reps = clampIntValue(6+int(v*15), 1, 30)
	}
	if v, ok := get(6); ok {
		params.RestSeconds = 30 + int(v*90)
	}
	if v, ok := get(7); ok {
		params.ExerciseCount = int(v * 6)
	}
	for i := 8; i < len(vec) && i < 19; i++ {
		params.SelectionWeights = append(params.SelectionWeights, clamp01(vec[i]))
	}
	return params
}

// selectExercises выбирает подмножество упражнений из пула по весам GAN.
// Если весов нет — возвращает весь пул. Число упражнений задаётся params.ExerciseCount.
func selectExercises(pool []string, params GANPlanParams) []string {
	if len(pool) == 0 || len(params.SelectionWeights) == 0 {
		return pool
	}
	type scoredExercise struct {
		name   string
		weight float64
		index  int
	}
	ranked := make([]scoredExercise, len(pool))
	for i, name := range pool {
		weight := params.SelectionWeights[i%len(params.SelectionWeights)]
		ranked[i] = scoredExercise{name: name, weight: weight, index: i}
	}
	sort.SliceStable(ranked, func(a, b int) bool {
		if ranked[a].weight != ranked[b].weight {
			return ranked[a].weight > ranked[b].weight
		}
		return ranked[a].index < ranked[b].index
	})
	count := len(pool)
	if params.ExerciseCount > 0 && params.ExerciseCount < count {
		count = params.ExerciseCount
	}
	selected := make([]string, 0, count)
	for i := 0; i < count; i++ {
		selected = append(selected, ranked[i].name)
	}
	return selected
}

// GeneratePlanFromGAN строит план тренировок, используя все 19 измерений
// GAN-вектора: интенсивность, длительность, частоту, подходы, повторения, отдых
// и выбор упражнений из пула класса. Если вектор пуст, используются значения по
// умолчанию (как в шаблоне).
func GeneratePlanFromGAN(
	classification string,
	ganVector []float64,
	constraints PlanConstraints,
	extras ...PersonalizationInput,
) (*entity.TrainingPlan, []*entity.PlanWeek, error) {
	tpl, ok := templates.GetTemplate(classification)
	if !ok {
		classification = "endurance_basic"
		tpl, _ = templates.GetTemplate(classification)
	}

	// Персонализация: возраст, ИМТ, менструальная фаза, ограничения,
	// аллергии (для плана), стресс, сон и биометрия с устройства.
	personalizationNotes := []string{}
	if len(extras) > 0 {
		ganVector, personalizationNotes = PersonalizeVector(ganVector, extras[0])
	}

	params := DecodeGANVector(ganVector)
	selected := selectExercises(tpl.Exercises, params)
	if len(selected) == 0 {
		selected = []string{"Разминка"}
	}

	availableDays := constraints.AvailableDays
	if len(availableDays) == 0 {
		availableDays = []int{1, 3, 5}
	}
	sort.Ints(availableDays)

	weeksCount := constraints.DurationWeeks
	if weeksCount <= 0 {
		weeksCount = 1
	}
	perDay := params.DurationMinutes
	if perDay <= 0 {
		perDay = 45
	}
	perExercise := perDay / len(selected)
	if perExercise < 1 {
		perExercise = 1
	}

	weeks := make([]*entity.PlanWeek, 0, weeksCount)
	for week := 1; week <= weeksCount; week++ {
		days := make([]*entity.PlanDay, 0, len(availableDays))
		for _, dayOfWeek := range availableDays {
			exercises := make([]*entity.PlanExercise, 0, len(selected))
			for i, name := range selected {
				exercises = append(exercises, &entity.PlanExercise{
					ExerciseName:    name,
					DurationMinutes: toInt32(perExercise),
					Intensity:       params.Intensity,
					Sets:            toInt32(params.Sets),
					Reps:            toInt32(params.Reps),
					RestSeconds:     toInt32(params.RestSeconds),
					SortOrder:       toInt32(i),
				})
			}
			days = append(days, &entity.PlanDay{
				DayOfWeek:            toInt32(dayOfWeek),
				IsRestDay:            false,
				TotalDurationMinutes: toInt32(perExercise * len(selected)),
				Exercises:            exercises,
			})
		}
		weeks = append(weeks, &entity.PlanWeek{
			WeekNumber:           toInt32(week),
			TotalTrainingDays:    toInt32(len(availableDays)),
			TotalDurationMinutes: toInt32(perDay * len(availableDays)),
			Days:                 days,
		})
	}

	plan := &entity.TrainingPlan{
		Classification: classification,
		DurationWeeks:  weeksCount,
		AvailableDays:  availableDays,
		PlanData: map[string]interface{}{
			"name":              tpl.NameRu,
			"class":             classification,
			"weeks":             weeksCount,
			"intensity":         params.Intensity,
			"generation_method": "gan",
			"gan_sessions":      params.SessionsPerWeek,
			"gan_duration":      params.DurationMinutes,
			"gan_sets":          params.Sets,
			"gan_reps":          params.Reps,
			"gan_rest":          params.RestSeconds,
			"gan_exercises":     len(selected),
			"personalization":   personalizationNotes,
		},
	}
	return plan, weeks, nil
}
