package chat

import (
	"fmt"
	"strings"
)

// DietProfile — особенности пользователя, влияющие на рацион:
// антропометрия, цель, аллергии, ограничения, тип питания и активность.
type DietProfile struct {
	Goal         string
	Allergies    []string
	Restrictions []string
	Diet         string
	Age          int
	Gender       string
	WeightKg     float64
	HeightCm     float64
	Activity     string
}

// safeMeal — вариант блюда без распространённых аллергенов.
type safeMeal struct {
	breakfast Meal
	lunch     Meal
	snack     Meal
	dinner    Meal
}

// allergenSafeMeals — нейтральные блюда, которые подходят при любой из
// перечисленных аллергий (яйца, молоко, глютен, орехи, морепродукты, соя).
var allergenSafeMeals = safeMeal{
	breakfast: meal(mealBreakfast, "08:00", "Рисовая каша на воде с ягодами", 8, 52, 6),
	lunch:     meal(mealLunch, "13:00", "Гречка с индейкой и тушёными овощами", 42, 45, 12),
	snack:     meal(mealSnack, "16:30", "Яблоко и горсть тыквенных семечек", 8, 28, 8),
	dinner:    meal(mealDinner, "19:30", "Запечённая индейка с овощами и оливковым маслом", 40, 18, 20),
}

// allergenPatterns сопоставляет аллерген с подстроками в описании блюда.
var allergenPatterns = []struct {
	names    []string
	patterns []string
}{
	{
		names:    []string{"орех", "nuts", "peanut", "арахис"},
		patterns: []string{"орех", "миндал", "фундук", "кешью", "грецк", "арахис", "pesto", "песто"},
	},
	{
		names:    []string{"молок", "milk", "dairy"},
		patterns: []string{"молок", "йогурт", "творог", "сыр", "сливк", "кефир", "сметан", "масло сливоч"},
	},
	{
		names:    []string{"яйц", "egg"},
		patterns: []string{"яйц", "омлет"},
	},
	{
		names:    []string{"глютен", "gluten"},
		patterns: []string{"хлеб", "паста", "макарон", "овсян", "тост", "булгур", "кускус", "пшенич", "ячмен", "спельт", "лаваш", "сухар", "печень"},
	},
	{
		names:    []string{"морепродукт", "seafood", "fish", "рыб"},
		patterns: []string{"рыб", "лосос", "тунец", "треск", "креветк", "морепродукт", "сельдь", "скумбр", "мидии"},
	},
	{
		names:    []string{"соя", "soy"},
		patterns: []string{"соя", "соев", "тофу"},
	},
}

// conflictsWithAllergens возвращает аллерген, который встречается в тексте.
func conflictsWithAllergens(text string, allergies []string) string {
	if len(allergies) == 0 {
		return ""
	}
	lower := strings.ToLower(text)
	for _, allergy := range allergies {
		a := strings.ToLower(strings.TrimSpace(allergy))
		if a == "" || a == "нет аллергий" || a == "no allergies" {
			continue
		}
		for _, entry := range allergenPatterns {
			for _, name := range entry.names {
				if strings.Contains(a, name) {
					for _, pattern := range entry.patterns {
						if strings.Contains(lower, pattern) {
							return allergy
						}
					}
				}
			}
		}
	}
	return ""
}

// estimateCalories считает суточную калорийность по Миффлину–Сан Жеору.
func estimateCalories(p DietProfile) int {
	if p.WeightKg <= 0 || p.HeightCm <= 0 || p.Age <= 0 {
		return 0
	}
	weight := p.WeightKg
	height := p.HeightCm
	age := float64(p.Age)

	bmr := 10*weight + 6.25*height - 5*age
	if isFemaleProfile(p.Gender) {
		bmr -= 161
	} else {
		bmr += 5
	}
	if bmr <= 0 {
		return 0
	}

	activity := 1.2
	switch strings.ToLower(strings.TrimSpace(p.Activity)) {
	case "light", "лёгкий", "легкий":
		activity = 1.375
	case "moderate", "средний":
		activity = 1.55
	case "high", "высокий":
		activity = 1.725
	case "athlete", "профессиональный":
		activity = 1.9
	}

	target := bmr * activity
	switch strings.ToLower(strings.TrimSpace(p.Goal)) {
	case "weight_loss", "loss", "похуд", "снижение веса":
		target *= 0.85
	case "muscle_gain", "gain", "мышц", "набор":
		target *= 1.1
	}
	return int(target / 50 * 50)
}

func isFemaleProfile(gender string) bool {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "female", "f", "жен", "женский":
		return true
	default:
		return false
	}
}

// BuildDayMenuForProfile строит меню с учётом аллергий и калорийности.
func BuildDayMenuForProfile(goal string, p DietProfile) DayMenu {
	menu := BuildDayMenu(goal)
	if len(menu.Meals) < 4 {
		return menu
	}
	if len(p.Allergies) == 0 && estimateCalories(p) == 0 {
		return menu
	}

	notes := make([]string, 0, 4)
	breakfast, lunch, snack, dinner := menu.Meals[0], menu.Meals[1], menu.Meals[2], menu.Meals[3]

	if allergen := conflictsWithAllergens(breakfast.Description, p.Allergies); allergen != "" {
		breakfast = allergenSafeMeals.breakfast
		notes = append(notes, fmt.Sprintf("Завтрак заменён: аллергия — %s", allergen))
	}
	if allergen := conflictsWithAllergens(lunch.Description, p.Allergies); allergen != "" {
		lunch = allergenSafeMeals.lunch
		notes = append(notes, fmt.Sprintf("Обед заменён: аллергия — %s", allergen))
	}
	if allergen := conflictsWithAllergens(snack.Description, p.Allergies); allergen != "" {
		snack = allergenSafeMeals.snack
		notes = append(notes, fmt.Sprintf("Перекус заменён: аллергия — %s", allergen))
	}
	if allergen := conflictsWithAllergens(dinner.Description, p.Allergies); allergen != "" {
		dinner = allergenSafeMeals.dinner
		notes = append(notes, fmt.Sprintf("Ужин заменён: аллергия — %s", allergen))
	}

	menu.Meals = []Meal{breakfast, lunch, snack, dinner}
	menu = recalculateMacros(menu)

	if target := estimateCalories(p); target > 0 {
		scale := float64(target) / float64(menu.TotalKcal)
		if scale > 0.5 && scale < 2.0 {
			menu = scaleMenu(menu, scale)
			notes = append(notes, fmt.Sprintf("Калорийность подстроена под профиль: ~%d ккал", target))
		}
	}

	if len(notes) > 0 {
		menu.Note = menu.Note + " Персонализация: " + strings.Join(notes, "; ") + "."
	}
	return menu
}

func recalculateMacros(menu DayMenu) DayMenu {
	var kcal, protein, carbs, fat int
	for _, m := range menu.Meals {
		kcal += m.Kcal
		protein += m.ProteinG
		carbs += m.CarbsG
		fat += m.FatG
	}
	menu.TotalKcal = kcal
	menu.ProteinG = protein
	menu.CarbsG = carbs
	menu.FatG = fat
	return menu
}

func scaleMenu(menu DayMenu, scale float64) DayMenu {
	for i := range menu.Meals {
		menu.Meals[i].ProteinG = int(float64(menu.Meals[i].ProteinG) * scale)
		menu.Meals[i].CarbsG = int(float64(menu.Meals[i].CarbsG) * scale)
		menu.Meals[i].FatG = int(float64(menu.Meals[i].FatG) * scale)
		menu.Meals[i].Kcal = kcal(menu.Meals[i].ProteinG, menu.Meals[i].CarbsG, menu.Meals[i].FatG)
	}
	return recalculateMacros(menu)
}

// RespondWithProfile — Respond с учётом профиля питания.
func RespondWithProfile(question string, profile DietProfile) (answer string, menu *DayMenu) {
	if IsMenuRequest(question) {
		goal := DetectGoal(question)
		if goal == "balance" && profile.Goal != "" {
			goal = profile.Goal
		}
		m := BuildDayMenuForProfile(goal, profile)
		return MenuSummary(m), &m
	}
	return FindAnswer(question), nil
}
