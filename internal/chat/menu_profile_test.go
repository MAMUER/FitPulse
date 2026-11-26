package chat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDayMenuForProfile_AllergyReplacesMeal(t *testing.T) {
	base := BuildDayMenu("balance")
	require.NotEmpty(t, base.Meals)

	profile := DietProfile{
		Goal:      "balance",
		Allergies: []string{"Молоко"},
		Age:       30,
		Gender:    "male",
		WeightKg:  75,
		HeightCm:  180,
	}

	menu := BuildDayMenuForProfile("balance", profile)

	// Молоко было в oatmeal/йогурте — завтрак и перекус должны быть заменены.
	assert.NotContains(t, strings.ToLower(menu.Meals[0].Description), "молок")
	assert.NotContains(t, strings.ToLower(menu.Meals[2].Description), "йогурт")
	assert.Contains(t, menu.Note, "Персонализация")
}

func TestBuildDayMenuForProfile_NoAllergiesKeepsTemplate(t *testing.T) {
	menu := BuildDayMenuForProfile("balance", DietProfile{})
	base := BuildDayMenu("balance")
	assert.Equal(t, base.Meals, menu.Meals)
	assert.Equal(t, base.TotalKcal, menu.TotalKcal)
}

func TestBuildDayMenuForProfile_ScalesCalories(t *testing.T) {
	// Женщина 30 лет, 60 кг, 165 см, сидячий образ жизни, цель — похудение.
	profile := DietProfile{
		Goal:     "weight_loss",
		Age:      30,
		Gender:   "female",
		WeightKg: 60,
		HeightCm: 165,
		Activity: "sedentary",
	}

	menu := BuildDayMenuForProfile("weight_loss", profile)
	target := estimateCalories(profile)
	require.Greater(t, target, 0)

	// П口径 допуск ±15%: шаблон масштабируется целиком.
	diff := menu.TotalKcal - target
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, target*15/100,
		"ожидалось ~%d ккал, получено %d", target, menu.TotalKcal)
	assert.Contains(t, menu.Note, "Калорийность")
}

func TestRespondWithProfile_MenuRequest(t *testing.T) {
	answer, menu := RespondWithProfile("Составь меню на день", DietProfile{
		Goal:      "muscle_gain",
		Allergies: []string{"Орехи"},
		Age:       28,
		Gender:    "male",
		WeightKg:  80,
		HeightCm:  182,
	})

	require.NotNil(t, menu)
	assert.NotEmpty(t, answer)
	assert.Equal(t, "muscle_gain", menu.Goal)
}

func TestRespondWithProfile_NonMenuRequest(t *testing.T) {
	answer, menu := RespondWithProfile("Как дела?", DietProfile{})
	assert.Nil(t, menu)
	assert.NotEmpty(t, answer)
}

func TestEstimateCalories_InsufficientData(t *testing.T) {
	assert.Equal(t, 0, estimateCalories(DietProfile{}))
}

func TestConflictsWithAllergens(t *testing.T) {
	assert.Equal(t, "Орехи", conflictsWithAllergens("Овсянка с миндалём", []string{"Орехи"}))
	assert.Equal(t, "Глютен", conflictsWithAllergens("Тост из цельнозернового хлеба", []string{"Глютен"}))
	assert.Equal(t, "Морепродукты", conflictsWithAllergens("Запечённый лосось", []string{"Морепродукты"}))
	assert.Empty(t, conflictsWithAllergens("Гречка с индейкой", []string{"Орехи", "Молоко"}))
	assert.Empty(t, conflictsWithAllergens("Что угодно", []string{"Нет аллергий"}))
}

func TestBuildDayMenuForProfile_MenstrualPhase(t *testing.T) {
	base := DietProfile{
		Goal: "balance", Age: 30, Gender: "female", WeightKg: 60, HeightCm: 168,
	}

	menses := base
	menses.MenstrualPhase = "menstruation"
	menu := BuildDayMenuForProfile("balance", menses)
	assert.Contains(t, menu.Note, "фаза цикла")

	luteal := base
	luteal.MenstrualPhase = "luteal"
	menu = BuildDayMenuForProfile("balance", luteal)
	assert.Contains(t, menu.Note, "фаза цикла")

	male := base
	male.Gender = "male"
	male.MenstrualPhase = "menstruation"
	menu = BuildDayMenuForProfile("balance", male)
	assert.NotContains(t, menu.Note, "менструация")
}

func TestBuildDayMenuForProfile_Stress(t *testing.T) {
	profile := DietProfile{Goal: "balance", Age: 30, Gender: "male", WeightKg: 75, HeightCm: 180, StressLevel: 8}
	menu := BuildDayMenuForProfile("balance", profile)
	assert.Contains(t, menu.Note, "стресс")
}

func TestBuildDayMenuForProfile_Diabetes(t *testing.T) {
	profile := DietProfile{
		Goal: "balance", Age: 55, Gender: "male", WeightKg: 90, HeightCm: 175,
		Restrictions: []string{"Сахарный диабет 2 типа"},
	}
	menu := BuildDayMenuForProfile("balance", profile)
	assert.Contains(t, menu.Note, "диабет")
}

func TestBuildDayMenuForProfile_UsesFoodDB(t *testing.T) {
	if FoodDBSize() == 0 {
		t.Skip("food_data.json не загружен")
	}
	profile := DietProfile{
		Goal: "balance", Age: 30, Gender: "male", WeightKg: 75, HeightCm: 180,
	}
	menu := BuildDayMenuForProfile("balance", profile)
	assert.Contains(t, menu.Note, "Open Food Facts")
	assert.Len(t, menu.Meals, 4)
	assert.Greater(t, menu.TotalKcal, 0)
}

func TestBuildDayMenuForProfile_AllergyAvoidsDish(t *testing.T) {
	if FoodDBSize() == 0 {
		t.Skip("food_data.json не загружен")
	}
	profile := DietProfile{
		Goal: "balance", Age: 30, Gender: "male", WeightKg: 75, HeightCm: 180,
		Allergies: []string{"Молоко", "Орехи", "Глютен", "Яйца"},
	}
	menu := BuildDayMenuForProfile("balance", profile)
	for _, m := range menu.Meals {
		assert.Empty(t, conflictsWithAllergens(m.Description, profile.Allergies),
			"блюдо %q конфликтует с аллергиями", m.Description)
	}
}
