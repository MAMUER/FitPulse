package chat

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFoodDBHelpers(t *testing.T) {
	old := foodDB
	t.Cleanup(func() { foodDB = old })

	t.Run("empty db", func(t *testing.T) {
		foodDB = nil
		_, ok := pickFoodItem(rand.New(rand.NewSource(1)), nil, 0, 100)
		assert.False(t, ok)
		_, ok = buildMenuFromFoodDB("balance", DietProfile{}, rand.New(rand.NewSource(1)))
		assert.False(t, ok)
		assert.Equal(t, 0, FoodDBSize())
	})

	t.Run("to meal", func(t *testing.T) {
		item := FoodItem{Name: "Тест", Kcal: 100, Protein: 5, Carbs: 10, Fat: 3}
		m := foodToMeal(item, mealBreakfast, "08:00")
		assert.Equal(t, "Тест", m.Description)
		assert.Equal(t, 100, m.Kcal)
		assert.Equal(t, "08:00", m.Time)
	})

	t.Run("allergen conflict", func(t *testing.T) {
		assert.True(t, foodAllergenConflict(FoodItem{Name: "Молоко", Allergens: "milk"}, []string{"Молоко"}))
		assert.False(t, foodAllergenConflict(FoodItem{Name: "Гречка"}, []string{"Орехи"}))
	})

	t.Run("has extras", func(t *testing.T) {
		assert.True(t, hasProfileExtrasForFood(DietProfile{Allergies: []string{"Орехи"}}))
		assert.True(t, hasProfileExtrasForFood(DietProfile{StressLevel: 8}))
		assert.False(t, hasProfileExtrasForFood(DietProfile{}))
	})
}

func TestApplyHealthRestrictions_Diabetes(t *testing.T) {
	snack := meal(mealSnack, "16:30", "Йогурт", 10, 20, 5)
	out, note := applyHealthRestrictions([]string{"Сахарный диабет 2 типа"}, snack)
	assert.Contains(t, note, "диабет")
	assert.NotEqual(t, "Йогурт", out.Description)
}

func TestApplyMenstrualNotes_Phases(t *testing.T) {
	base := DietProfile{Gender: "female"}

	menses := base
	menses.MenstrualPhase = "menstruation"
	lunch := meal(mealLunch, "13:00", "Обед", 30, 40, 10)
	assert.Contains(t, applyMenstrualNotes(menses, &lunch), "менструация")

	luteal := base
	luteal.MenstrualPhase = "luteal"
	assert.Contains(t, applyMenstrualNotes(luteal, &lunch), "лютеиновая")

	follicular := base
	follicular.MenstrualPhase = "follicular"
	assert.Contains(t, applyMenstrualNotes(follicular, &lunch), "благоприятная")

	male := base
	male.Gender = "male"
	male.MenstrualPhase = "menstruation"
	assert.Empty(t, applyMenstrualNotes(male, &lunch))
}

func TestApplyStressNotes(t *testing.T) {
	snack := meal(mealSnack, "16:30", "Яблоко", 5, 20, 2)
	assert.Contains(t, applyStressNotes(8, &snack), "стресс")
	assert.Empty(t, applyStressNotes(3, &snack))
}

func TestBuildDayMenuForProfile_MenstrualAndStress(t *testing.T) {
	p := DietProfile{
		Goal: "balance", Age: 28, Gender: "female", WeightKg: 60, HeightCm: 168,
		MenstrualPhase: "menstruation", StressLevel: 8,
	}
	menu := BuildDayMenuForProfile("balance", p)
	assert.NotEmpty(t, menu.Note)
	assert.Len(t, menu.Meals, 4)
}
