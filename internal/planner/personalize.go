package planner

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MAMUER/project/internal/domain/entity"
)

// PersonalizationInput — все особенности пользователя, которые влияют на план:
// антропометрия, цель, менструальный цикл, ограничения по здоровью, аллергии,
// стресс, сон и биометрия с устройства.
//
// Классификатор определяет ТОЛЬКО функциональное состояние по физиологии
// (ЧСС/HRV/SpO2/температура). Всё остальное применяется здесь, при сборке
// плана, потому что на этих признаках нельзя обучить модель состояния — но
// они обязаны менять нагрузку и рацион.
type PersonalizationInput struct {
	Age            int
	Gender         string
	WeightKg       float64
	HeightCm       float64
	Goal           string
	FitnessLevel   string
	MenstrualPhase string // "", "menstruation", "follicular", "ovulation", "luteal"
	FlowIntensity  string
	StressLevel    float64 // 0..10, отрицательное = неизвестно
	SleepHours     float64
	Allergies      []string
	Restrictions   []string // "Диабет", "Гипертония", ...
	Biometrics     *Biometrics
}

// Adjustment — итог персонализации: множители и человекочитаемые пояснения.
type Adjustment struct {
	IntensityFactor float64
	IntensityCap    float64 // >0 — абсолютный потолок интенсивности (0 = нет)
	DurationFactor  float64
	SessionsDelta   int
	SetsFactor      float64
	Notes           []string
}

// clampUnit ограничивает значение диапазоном [0,1].
func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (in PersonalizationInput) female() bool {
	g := strings.ToLower(strings.TrimSpace(in.Gender))
	return g == "female" || g == "f" || g == "жен" || g == "женский"
}

// BMI возвращает индекс массы тела и признак его доступности.
func (in PersonalizationInput) BMI() (float64, bool) {
	if in.WeightKg <= 0 || in.HeightCm <= 0 {
		return 0, false
	}
	h := in.HeightCm / 100.0
	return in.WeightKg / (h * h), true
}

// noteFunc добавляет человекочитаемое пояснение.
type noteFunc func(format string, args ...interface{})

// Personalize вычисляет модификаторы нагрузки по всем доступным данным.
// Правила консервативные: каждое ограничение только снижает нагрузку, чтобы
// не навредить пользователю с заболеванием или в фазе цикла.
func Personalize(in PersonalizationInput) Adjustment {
	adj := Adjustment{
		IntensityFactor: 1.0,
		DurationFactor:  1.0,
		SetsFactor:      1.0,
	}
	note := func(format string, args ...interface{}) {
		adj.Notes = append(adj.Notes, fmt.Sprintf(format, args...))
	}

	applyAge(&adj, in.Age, note)
	applyBMI(&adj, in, note)
	applyMenstrualPhase(&adj, in, note)
	applyStress(&adj, in.StressLevel, note)
	applySleep(&adj, in.SleepHours, in.Biometrics, note)
	applyBiometrics(&adj, in.Biometrics, note)
	applyRestrictions(&adj, in.Restrictions, note)
	applyGoal(&adj, in.Goal)
	return finalizeAdjustment(adj)
}

func applyAge(adj *Adjustment, age int, note noteFunc) {
	switch {
	case age > 0 && age < 18:
		adj.IntensityFactor *= 0.85
		note("возраст до 18 лет: интенсивность ×0.85")
	case age >= 75:
		adj.IntensityFactor *= 0.65
		adj.SetsFactor *= 0.8
		adj.SessionsDelta--
		note("возраст 75+: интенсивность ×0.65, меньше подходов")
	case age > 60:
		adj.IntensityFactor *= 0.8
		note("возраст 60+: интенсивность ×0.8")
	}
}

func applyBMI(adj *Adjustment, in PersonalizationInput, note noteFunc) {
	bmi, ok := in.BMI()
	if !ok {
		return
	}
	switch {
	case bmi < 18.5:
		adj.IntensityFactor *= 0.9
		note("ИМТ %.1f (недостаток массы): интенсивность ×0.9", bmi)
	case bmi >= 35:
		adj.IntensityFactor *= 0.7
		adj.SetsFactor *= 0.85
		note("ИМТ %.1f (ожирение): интенсивность ×0.7", bmi)
	case bmi >= 30:
		adj.IntensityFactor *= 0.85
		note("ИМТ %.1f: интенсивность ×0.85", bmi)
	case bmi >= 25:
		adj.IntensityFactor *= 0.95
		note("ИМТ %.1f (избыточная масса): интенсивность ×0.95", bmi)
	}
}

func applyMenstrualPhase(adj *Adjustment, in PersonalizationInput, note noteFunc) {
	if !in.female() {
		return
	}
	switch strings.ToLower(strings.TrimSpace(in.MenstrualPhase)) {
	case "menstruation", "menstrual", "менструация":
		adj.IntensityFactor *= 0.8
		note("менструация: интенсивность ×0.8")
	case "luteal", "лютеиновая":
		adj.IntensityFactor *= 0.9
		note("лютеиновая фаза: интенсивность ×0.9")
	case "follicular", "фолликулярная", "ovulation", "овуляция":
		// Фаза, в которой организм переносит нагрузку лучше всего —
		// специальных снижений нет, фиксируем в примечании.
		note("благоприятная фаза цикла: без снижения нагрузки")
	}
	if strings.EqualFold(in.FlowIntensity, "heavy") || in.FlowIntensity == "Обильные" {
		adj.IntensityFactor *= 0.9
		note("обильные выделения: дополнительно ×0.9")
	}
}

func applyStress(adj *Adjustment, stress float64, note noteFunc) {
	if stress < 0 {
		return
	}
	switch {
	case stress >= 7:
		adj.IntensityFactor *= 0.75
		adj.DurationFactor *= 0.9
		note("высокий уровень стресса %.1f/10: интенсивность ×0.75", stress)
	case stress >= 5:
		adj.IntensityFactor *= 0.9
		note("умеренный стресс %.1f/10: интенсивность ×0.9", stress)
	}
}

func applySleep(adj *Adjustment, hours float64, biometrics *Biometrics, note noteFunc) {
	sleep := hours
	if sleep <= 0 && biometrics != nil {
		sleep = biometrics.SleepHours
	}
	switch {
	case sleep > 0 && sleep < 5:
		adj.IntensityFactor *= 0.7
		adj.DurationFactor *= 0.9
		note("сон %.1f ч: интенсивность ×0.7", sleep)
	case sleep > 0 && sleep < 6.5:
		adj.IntensityFactor *= 0.85
		note("сон %.1f ч: интенсивность ×0.85", sleep)
	}
}

func applyBiometrics(adj *Adjustment, b *Biometrics, note noteFunc) {
	if b == nil {
		return
	}
	if b.HRV > 0 && b.HRV < 20 {
		adj.IntensityCap = 0.4
		adj.IntensityFactor *= 0.5
		note("HRV %.0f мс (низкая): интенсивность ≤0.4", b.HRV)
	} else if b.HRV > 0 && b.HRV < 40 {
		adj.IntensityFactor *= 0.85
		note("HRV %.0f мс (снижена): ×0.85", b.HRV)
	}
	if b.HeartRate > 100 {
		adj.IntensityFactor *= 0.7
		note("ЧСС покоя %.0f: интенсивность ×0.7", b.HeartRate)
	} else if b.HeartRate > 85 {
		adj.IntensityFactor *= 0.9
		note("ЧСС покоя %.0f: интенсивность ×0.9", b.HeartRate)
	}
	if b.Spo2 > 0 && b.Spo2 < 94 {
		adj.IntensityFactor *= 0.7
		note("SpO2 %.0f%%: интенсивность ×0.7", b.Spo2)
	}
	if b.Temperature > 37.5 {
		adj.IntensityFactor *= 0.6
		adj.DurationFactor *= 0.8
		note("температура %.1f°C: нагрузка снижена", b.Temperature)
	}
}

func applyRestrictions(adj *Adjustment, restrictions []string, note noteFunc) {
	for _, raw := range restrictions {
		r := strings.ToLower(strings.TrimSpace(raw))
		switch {
		case strings.Contains(r, "сердеч"), strings.Contains(r, "heart"):
			adj.IntensityFactor *= 0.6
			note("сердечное заболевание: интенсивность ×0.6")
		case strings.Contains(r, "гипертония"), strings.Contains(r, "hypertension"):
			adj.IntensityFactor *= 0.75
			note("гипертония: интенсивность ×0.75")
		case strings.Contains(r, "диабет"), strings.Contains(r, "diabetes"):
			adj.IntensityFactor *= 0.85
			note("диабет: интенсивность ×0.85")
		case strings.Contains(r, "почеч"), strings.Contains(r, "почек"), strings.Contains(r, "kidney"):
			adj.IntensityFactor *= 0.8
			note("заболевание почек: интенсивность ×0.8")
		case strings.Contains(r, "печён"), strings.Contains(r, "печен"), strings.Contains(r, "liver"):
			adj.IntensityFactor *= 0.8
			note("заболевание печени: интенсивность ×0.8")
		case strings.Contains(r, "ортопед"), strings.Contains(r, "orthopedic"):
			adj.IntensityFactor *= 0.8
			note("ортопедическое ограничение: интенсивность ×0.8")
		}
	}
}

func applyGoal(adj *Adjustment, goal string) {
	switch strings.ToLower(strings.TrimSpace(goal)) {
	case "weight_loss", "loss", "похуд", "снижение веса":
		adj.IntensityFactor *= 0.95
	case "muscle_gain", "gain", "мышц", "набор":
		// Профицит калорий и силовая работа — без снижения.
	default:
	}
}

// finalizeAdjustment ограничивает множители допустимым диапазоном.
func finalizeAdjustment(adj Adjustment) Adjustment {
	adj.IntensityFactor = clampUnit(adj.IntensityFactor)
	adj.DurationFactor = clampUnit(adj.DurationFactor)
	adj.SetsFactor = clampUnit(adj.SetsFactor)
	if adj.SetsFactor < 0.5 {
		adj.SetsFactor = 0.5
	}
	return adj
}

// PersonalizeVector применяет персонализацию к 19-мерному вектору GAN.
// Длины вектора менять нельзя — генератор и денормализация ожидают 19.
func PersonalizeVector(vec []float64, in PersonalizationInput) ([]float64, []string) {
	if len(vec) == 0 {
		adj := Personalize(in)
		return vec, adj.Notes
	}
	adj := Personalize(in)
	out := make([]float64, len(vec))
	copy(out, vec)

	// dim0 — интенсивность, dim1 — длительность, dim2 — частота,
	// dim4 — подходы. Остальные измерения (упражнения, отдых) не трогаем:
	// их семантика задана моделью, а ограничения ниже уже меняют нагрузку.
	out[0] = clampUnit(out[0] * adj.IntensityFactor)
	if adj.IntensityCap > 0 && out[0] > adj.IntensityCap {
		out[0] = adj.IntensityCap
	}
	if len(out) > 1 {
		out[1] = clampUnit(out[1] * adj.DurationFactor)
	}
	if len(out) > 2 && adj.SessionsDelta != 0 {
		out[2] = clampUnit(out[2] + float64(adj.SessionsDelta)*0.15)
	}
	if len(out) > 4 {
		out[4] = clampUnit(out[4] * adj.SetsFactor)
	}
	return out, adj.Notes
}

// stringSliceFromAny приводит значение опроса (JSON) к []string.
// Поддерживает []interface{}, []string и строку с разделителем.
func stringSliceFromAny(v interface{}) []string {
	switch typed := v.(type) {
	case nil:
		return nil
	case []string:
		return typed
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		parts := strings.FieldsFunc(typed, func(r rune) bool {
			return r == ',' || r == ';' || r == '|'
		})
		return parts
	default:
		return nil
	}
}

// firstNumberFromSurvey вытаскивает первое число из ответа «7–8 ч».
func firstNumberFromSurvey(v interface{}) float64 {
	s, ok := v.(string)
	if !ok {
		if f, ok := v.(float64); ok {
			return f
		}
		return 0
	}
	s = strings.ReplaceAll(s, ",", ".")
	// Берём первую последовательность цифр.
	start := -1
	for i, r := range s {
		if r >= '0' && r <= '9' {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			if val, err := strconv.ParseFloat(s[start:i], 64); err == nil {
				return val
			}
			start = -1
		}
	}
	if start >= 0 {
		if val, err := strconv.ParseFloat(s[start:], 64); err == nil {
			return val
		}
	}
	return 0
}

// PersonalizationFromUser собирает вход персонализации из профиля, опроса и
// биометрии. Это единственное место, где «вес/возраст/цикл/аллергии/стресс»
// попадают в генерацию плана.
func PersonalizationFromUser(
	user *entity.User,
	survey map[string]interface{},
	biometrics *Biometrics,
) PersonalizationInput {
	in := PersonalizationInput{Biometrics: biometrics, StressLevel: -1}
	if user != nil {
		in.Age = int(user.Age)
		in.Gender = user.Gender
		in.WeightKg = user.WeightKg
		in.HeightCm = float64(user.HeightCm)
		in.FitnessLevel = user.FitnessLevel
		in.SleepHours = float64(user.SleepHours)
		if len(user.Goals) > 0 {
			in.Goal = strings.ToLower(user.Goals[0])
		}
	}

	if survey != nil {
		in.Allergies = stringSliceFromAny(survey["allergies"])
		in.Restrictions = stringSliceFromAny(survey["restrictions"])
		if in.Goal == "" {
			in.Goal = surveyString(survey, "goal", "goals")
		}
		if sh := firstNumberFromSurvey(survey["sleepHours"]); sh > 0 {
			in.SleepHours = sh
		}
		in.MenstrualPhase = surveyString(survey, "menstrualPhase", "menstrual_phase", "cyclePhase")
		in.FlowIntensity = surveyString(survey, "flowIntensity", "flow_intensity")
		in.StressLevel = firstNumberFromSurvey(survey["stress"])
		if in.StressLevel <= 0 {
			in.StressLevel = firstNumberFromSurvey(survey["stressLevel"])
		}
		if in.StressLevel <= 0 && surveyGoalMentionsStress(survey["goals"]) {
			// «Снизить стресс» — косвенный признак повышенного стресса.
			in.StressLevel = 6
		}
	}

	return in
}

func surveyString(survey map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := survey[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

func surveyGoalMentionsStress(v interface{}) bool {
	for _, item := range stringSliceFromAny(v) {
		if strings.Contains(strings.ToLower(item), "стресс") || strings.Contains(strings.ToLower(item), "stress") {
			return true
		}
	}
	return false
}

// MenstrualPhaseFromCycle вычисляет фазу цикла по дате начала и длительности.
// Модель 28 дней: 1-5 менструация, 6-12 фолликулярная, 13-16 овуляция,
// 17-28 лютеиновая. Если даты нет — возвращает пустую строку.
func MenstrualPhaseFromCycle(startDate string, now time.Time) (phase string, flow string) {
	if strings.TrimSpace(startDate) == "" {
		return "", ""
	}
	trimmed := strings.TrimSpace(startDate)
	if len(trimmed) < 10 {
		return "", ""
	}
	start, err := time.Parse("2006-01-02", trimmed[:10])
	if err != nil {
		return "", ""
	}
	day := int(now.Sub(start).Hours()/24) + 1
	if day < 1 {
		return "", ""
	}
	switch {
	case day <= 5:
		return "menstruation", "heavy"
	case day <= 12:
		return "follicular", "medium"
	case day <= 16:
		return "ovulation", "medium"
	default:
		return "luteal", "light"
	}
}
