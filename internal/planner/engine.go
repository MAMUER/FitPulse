// Package planner generates training plans from templates.
package planner

import (
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
	HRV          float64
	HeartRate    float64
	Spo2         float64
	SleepHours   float64
	Temperature  float64
	SystolicBP   float64
	DiastolicBP  float64
}

func GeneratePlanFromTemplate(
	classification string,
	user *entity.User,
	survey map[string]interface{},
	biometrics *Biometrics,
	constraints PlanConstraints,
) (*entity.TrainingPlan, []*entity.PlanWeek, error) {
	tpl, ok := templates.GetTemplate(classification)
	if !ok {
		classification = "endurance_basic"
		tpl, _ = templates.GetTemplate(classification)
	}

	intensity := applyModifiers(tpl.IntensityRange[0], user, survey, biometrics)
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
				DurationMinutes: int32(duration / len(availableDays) / exerciseCount),
				Intensity:       intensity,
				Sets:            3,
				Reps:            12,
				RestSeconds:     int32(float64(duration/len(availableDays)) * tpl.RestRatio),
				SortOrder:       int32(exIdx),
			})
			}
			days = append(days, &entity.PlanDay{
				DayOfWeek:            int32(dayOfWeek),
				IsRestDay:            false,
				TotalDurationMinutes: int32(duration / len(availableDays)),
				Notes:                "",
				Exercises:            exercises,
			})
		}
		weeks = append(weeks, &entity.PlanWeek{
			WeekNumber:        int32(week),
			TotalTrainingDays: int32(len(availableDays)),
			TotalDurationMinutes: int32(duration / constraints.DurationWeeks),
			Days:              days,
		})
	}

	plan := &entity.TrainingPlan{
		Classification: classification,
		DurationWeeks:  constraints.DurationWeeks,
		AvailableDays:  availableDays,
		PlanData: map[string]interface{}{
			"name":    tpl.NameRu,
			"class":   classification,
			"weeks":   len(weeks),
			"intensity": intensity,
		},
	}
	return plan, weeks, nil
}

func applyModifiers(baseIntensity float64, user *entity.User, survey map[string]interface{}, biometrics *Biometrics) float64 {
	intensity := baseIntensity

	if user.Age > 60 {
		intensity *= 0.8
	}

	if bmi := float64(user.WeightKg) / math.Pow(float64(user.HeightCm)/100, 2); bmi > 35 {
		intensity *= 0.7
	}

	if survey != nil {
		if menstruation, ok := survey["menstruation"].(bool); ok && menstruation {
			intensity *= 0.85
		}
	}

	if biometrics != nil {
		if biometrics.HRV < 20 {
			intensity = math.Min(intensity, 0.4)
		}
		if biometrics.SleepHours > 0 && biometrics.SleepHours < 5 {
			intensity *= 0.7
		}
	}

	return math.Max(0.1, math.Min(1.0, intensity))
}
