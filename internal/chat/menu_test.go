package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsMenuRequest(t *testing.T) {
	assert.True(t, IsMenuRequest("Составь меню на сегодня"))
	assert.True(t, IsMenuRequest("Замени ужин на лёгкий"))
	assert.True(t, IsMenuRequest("что приготовить на обед"))
	assert.True(t, IsMenuRequest("make a menu for today"))
	assert.False(t, IsMenuRequest("Как изменить план тренировок?"))
	assert.False(t, IsMenuRequest("xyz"))
}

func TestDetectGoal(t *testing.T) {
	assert.Equal(t, "weight_loss", DetectGoal("хочу похудеть и сбросить вес"))
	assert.Equal(t, "muscle_gain", DetectGoal("набор мышечной массы"))
	assert.Equal(t, "light_dinner", DetectGoal("замени ужин на лёгкий"))
	assert.Equal(t, "balance", DetectGoal("просто меню на сегодня"))
	// Unknown goal falls back to balance.
	assert.Equal(t, "balance", DetectGoal("что-то непонятное"))
}

func TestBuildDayMenuDeterministic(t *testing.T) {
	m1 := BuildDayMenu("balance")
	m2 := BuildDayMenu("balance")
	assert.Equal(t, m1, m2, "same goal must yield the same menu")

	assert.Equal(t, "balance", m1.Goal)
	assert.NotEmpty(t, m1.Meals)
	assert.Equal(t, 4, len(m1.Meals))

	// Totals must equal the sum of meals.
	var kcal, p, c, f int
	for _, meal := range m1.Meals {
		kcal += meal.Kcal
		p += meal.ProteinG
		c += meal.CarbsG
		f += meal.FatG
		// Each meal's kcal must be consistent with its macros.
		assert.Equal(t, meal.ProteinG*4+meal.CarbsG*4+meal.FatG*9, meal.Kcal)
	}
	assert.Equal(t, kcal, m1.TotalKcal)
	assert.Equal(t, p, m1.ProteinG)
	assert.Equal(t, c, m1.CarbsG)
	assert.Equal(t, f, m1.FatG)
	assert.Greater(t, m1.TotalKcal, 0)
	assert.Greater(t, m1.WaterMl, 0)
}

func TestBuildDayMenuUnknownGoalFallsBack(t *testing.T) {
	m := BuildDayMenu("does_not_exist")
	assert.Equal(t, "balance", m.Goal)
}

func TestBuildDayMenuLightDinner(t *testing.T) {
	m := BuildDayMenu("light_dinner")
	assert.Equal(t, "light_dinner", m.Goal)
	// The dinner meal must be marked light.
	assert.Contains(t, m.Meals[3].Name, "Ужин")
	assert.Contains(t, m.Meals[3].Description, "тунц")
}

func TestMenuSummary(t *testing.T) {
	m := BuildDayMenu("weight_loss")
	summary := MenuSummary(m)
	assert.Contains(t, summary, "Составил меню на сегодня")
	assert.Contains(t, summary, "Завтрак")
	assert.Contains(t, summary, "Ужин")
	assert.Contains(t, summary, "Вода")
}

func TestRespond_Menu(t *testing.T) {
	answer, menu := Respond("Составь меню на сегодня")
	assert.NotNil(t, menu)
	assert.Contains(t, answer, "Составил меню")
	assert.Equal(t, "balance", menu.Goal)
}

func TestRespond_FAQFallback(t *testing.T) {
	answer, menu := Respond("Как изменить план?")
	assert.Nil(t, menu)
	assert.Equal(t, FindAnswer("Как изменить план?"), answer)
}
