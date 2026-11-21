// Package planner generates training plans from templates.
package planner

import (
	"fmt"
	"math"
	"sort"

	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/templates"
)

type PlanConstraints struct {
	DurationWeeks int
	AvailableDays []int
}

type Biometrics struct {
	HRV         float64
	HeartRate   float64
	Spo2        float64
	SleepHours  float64
	Temperature float64
	SystolicBP  float64
	DiastolicBP float64
}

func toInt32(v int) int32 {
	if v < math.MinInt32 || v > math.MaxInt32 {
		panic(fmt.Sprintf("planner: value %d overflows int32", v))
	}
	return int32(v)
}

func GeneratePlanFromTemplate(
	classification string,
	user *entity.User,
	survey map[string]interface{},
	biometrics *Biometrics,
	constraints PlanConstraints,
	extras ...PersonalizationInput,
) (*entity.TrainingPlan, []*entity.PlanWeek, error) {
	tpl, ok := templates.GetTemplate(classification)
	if !ok {
		classification = "endurance_basic"
		tpl, _ = templates.GetTemplate(classification)
	}

	personalization := PersonalizationFromUser(user, survey, biometrics)
	if len(extras) > 0 {
		personalization = extras[0]
	}
	adj := Personalize(personalization)
	intensity := applyModifiers(tpl.IntensityRange[0], adj)
	duration := tpl.DurationRange[1]
	if constraints.DurationWeeks > 0 {
		duration = constraints.DurationWeeks * 7
	}

	availableDays := constraints.AvailableDays
	if len(availableDays) == 0 {
		availableDays = []int{1, 3, 5}
	}

	sort.Ints(availableDays)
	weeks := make([]*entity.PlanWeek, 0, constraints.DurationWeeks)
	for week := 1; week <= constraints.DurationWeeks; week++ {
		days := make([]*entity.PlanDay, 0, len(availableDays))
		for _, dayOfWeek := range availableDays {
			exerciseCount := len(tpl.Exercises)
			if exerciseCount == 0 {
				exerciseCount = 1
			}
			exercises := make([]*entity.PlanExercise, 0, exerciseCount)
			for exIdx, exName := range tpl.Exercises {
				exercises = append(exercises, &entity.PlanExercise{
					ExerciseName:    exName,
					DurationMinutes: toInt32(duration / len(availableDays) / exerciseCount),
					Intensity:       intensity,
					Sets:            3,
					Reps:            12,
					RestSeconds:     toInt32(int(float64(duration/len(availableDays)) * tpl.RestRatio)),
					SortOrder:       toInt32(exIdx),
				})
			}
			days = append(days, &entity.PlanDay{
				DayOfWeek:            toInt32(dayOfWeek),
				IsRestDay:            false,
				TotalDurationMinutes: toInt32(duration / len(availableDays)),
				Notes:                "",
				Exercises:            exercises,
			})
		}
		weeks = append(weeks, &entity.PlanWeek{
			WeekNumber:           toInt32(week),
			TotalTrainingDays:    toInt32(len(availableDays)),
			TotalDurationMinutes: toInt32(duration / constraints.DurationWeeks),
			Days:                 days,
		})
	}

	plan := &entity.TrainingPlan{
		Classification: classification,
		DurationWeeks:  constraints.DurationWeeks,
		AvailableDays:  availableDays,
		PlanData: map[string]interface{}{
			"name":            tpl.NameRu,
			"class":           classification,
			"weeks":           len(weeks),
			"intensity":       intensity,
			"personalization": adj.Notes,
		},
	}
	return plan, weeks, nil
}

func applyModifiers(baseIntensity float64, adj Adjustment) float64 {
	intensity := baseIntensity * adj.IntensityFactor
	if adj.IntensityCap > 0 && intensity > adj.IntensityCap {
		intensity = adj.IntensityCap
	}
	return math.Max(0.1, math.Min(1.0, intensity))
}
