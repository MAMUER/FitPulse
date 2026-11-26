package chat

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
)

//go:embed food_data.json
var foodData []byte

// FoodItem — продукт из Open Food Facts / USDA (на 100 г).
type FoodItem struct {
	Name      string `json:"name"`
	Kcal      int    `json:"kcal"`
	Protein   int    `json:"protein"`
	Carbs     int    `json:"carbs"`
	Fat       int    `json:"fat"`
	Allergens string `json:"allergens"`
}

var foodDB = loadFoodDB()

func loadFoodDB() []FoodItem {
	var items []FoodItem
	if err := json.Unmarshal(foodData, &items); err != nil {
		return nil
	}
	return items
}

// FoodDBSize возвращает число доступных блюд (для тестов и диагностики).
func FoodDBSize() int { return len(foodDB) }

func foodAllergenConflict(item FoodItem, allergies []string) bool {
	return conflictsWithAllergens(item.Name+" "+item.Allergens, allergies) != ""
}

// pickFoodItem выбирает блюдо, подходящее по калориям и не конфликтующее с аллергиями.
func pickFoodItem(rng *rand.Rand, allergies []string, minKcal, maxKcal int) (FoodItem, bool) {
	if len(foodDB) == 0 {
		return FoodItem{}, false
	}
	start := rng.Intn(len(foodDB))
	for i := 0; i < len(foodDB); i++ {
		item := foodDB[(start+i)%len(foodDB)]
		if item.Kcal < minKcal || item.Kcal > maxKcal {
			continue
		}
		if foodAllergenConflict(item, allergies) {
			continue
		}
		return item, true
	}
	return FoodItem{}, false
}

func foodToMeal(item FoodItem, slot, time string) Meal {
	return Meal{
		Name:        slot,
		Time:        time,
		Description: item.Name,
		Kcal:        item.Kcal,
		ProteinG:    item.Protein,
		CarbsG:      item.Carbs,
		FatG:        item.Fat,
	}
}

// buildMenuFromFoodDB строит меню из реальных продуктов (если датасет загружен).
func buildMenuFromFoodDB(goal string, p DietProfile, rng *rand.Rand) (DayMenu, bool) {
	if len(foodDB) == 0 {
		return DayMenu{}, false
	}

	target := estimateCalories(p)
	if target <= 0 {
		target = 2000
	}

	slots := []struct {
		typ   string
		time  string
		share [2]float64
	}{
		{mealBreakfast, "08:00", [2]float64{0.20, 0.30}},
		{mealLunch, "13:00", [2]float64{0.30, 0.40}},
		{mealSnack, "16:30", [2]float64{0.10, 0.15}},
		{mealDinner, "19:30", [2]float64{0.20, 0.30}},
	}

	meals := make([]Meal, 0, 4)
	for _, slot := range slots {
		minKcal := int(float64(target) * slot.share[0])
		maxKcal := int(float64(target) * slot.share[1])
		item, ok := pickFoodItem(rng, p.Allergies, minKcal, maxKcal)
		if !ok {
			return DayMenu{}, false
		}
		meals = append(meals, foodToMeal(item, slot.typ, slot.time))
	}

	menu := recalculateMacros(DayMenu{
		Goal:  goal,
		Meals: meals,
		Note:  "Меню собрано из базы Open Food Facts / USDA (с учётом аллергий и калорийности).",
	})
	if target > 0 {
		scale := float64(target) / float64(menu.TotalKcal)
		if scale > 0.5 && scale < 2.0 {
			menu = scaleMenu(menu, scale)
			menu.Note += fmt.Sprintf(" Калорийность подстроена под профиль: ~%d ккал.", target)
		}
	}
	return menu, true
}

// hasProfileExtrasForFood — нужна ли персонализация для меню.
func hasProfileExtrasForFood(p DietProfile) bool {
	return len(p.Allergies) > 0 || estimateCalories(p) > 0 || p.MenstrualPhase != "" || p.StressLevel > 0
}

// hashString — детерминированный хеш для выбора блюд (стабильный между запусками).
func hashString(s string) int64 {
	var h int64 = 1469598103934665603
	for _, c := range s {
		h ^= int64(c)
		h *= 1099511628211
	}
	return h
}
