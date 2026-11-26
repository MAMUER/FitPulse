package chat

import (
	"fmt"
	"math/rand"
	"strings"
)

// DietProfile — особенности пользователя, влияющие на рацион:
// антропометрия, цель, аллергии, ограничения, тип питания и активность.
type DietProfile struct {
	Goal           string
	Allergies      []string
	Restrictions   []string
	Diet           string
	Age            int
	Gender         string
	WeightKg       float64
	HeightCm       float64
	Activity       string
	MenstrualPhase string
	StressLevel    float64
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
	lower := strings.ToLower(text)
	for _, allergy := range allergies {
		if conflictForAllergy(lower, allergy) != "" {
			return allergy
		}
	}
	return ""
}

// conflictForAllergy проверяет один аллерген против текста блюда.
func conflictForAllergy(lower, allergy string) string {
	a := strings.ToLower(strings.TrimSpace(allergy))
	if a == "" || a == "нет аллергий" || a == "no allergies" {
		return ""
	}
	for _, entry := range allergenPatterns {
		if matchesAny(a, entry.names) && matchesAny(lower, entry.patterns) {
			return allergy
		}
	}
	return ""
}

// matchesAny возвращает true, если строка содержит хотя бы один шаблон.
func matchesAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
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

// BuildDayMenuForProfile строит меню с учётом аллергий, калорийности,
// менструальной фазы, стресса и ограничений по здоровью.
func BuildDayMenuForProfile(goal string, p DietProfile) DayMenu {
	// Если есть профиль и загружена база продуктов — строим меню из реальных блюд.
	if hasProfileExtrasForFood(p) {
		rng := rand.New(rand.NewSource(hashString(goal + "|" + p.Goal + "|" + p.Diet)))
		if menu, ok := buildMenuFromFoodDB(goal, p, rng); ok {
			notes := make([]string, 0, 4)
			if len(p.Allergies) > 0 {
				notes = append(notes, "аллергии: "+strings.Join(p.Allergies, ", "))
			}
			if p.MenstrualPhase != "" {
				notes = append(notes, "фаза цикла: "+p.MenstrualPhase)
			}
			if p.StressLevel >= 7 {
				notes = append(notes, fmt.Sprintf("высокий стресс %.0f/10", p.StressLevel))
			}
			if len(p.Restrictions) > 0 {
				notes = append(notes, "ограничения: "+strings.Join(p.Restrictions, ", "))
			}
			if len(notes) > 0 {
				menu.Note += " Персонализация: " + strings.Join(notes, "; ") + "."
			}
			return menu
		}
	}

	menu := BuildDayMenu(goal)
	if len(menu.Meals) < 4 {
		return menu
	}
	if !hasProfileExtras(p) {
		return menu
	}

	notes := make([]string, 0, 6)
	notes = append(notes, replaceAllergens(&menu, p)...)
	notes = append(notes, applyRestrictionsAndPhase(&menu, p)...)
	menu = recalculateMacros(menu)
	notes = append(notes, scaleToCalories(&menu, p)...)

	if len(notes) > 0 {
		menu.Note = menu.Note + " Персонализация: " + strings.Join(notes, "; ") + "."
	}
	return menu
}

// replaceAllergens заменяет блюда, содержащие аллергены пользователя.
func replaceAllergens(menu *DayMenu, p DietProfile) []string {
	if len(p.Allergies) == 0 {
		return nil
	}
	var notes []string
	slots := []struct {
		name  string
		safe  Meal
		index int
	}{
		{"Завтрак", allergenSafeMeals.breakfast, 0},
		{"Обед", allergenSafeMeals.lunch, 1},
		{"Перекус", allergenSafeMeals.snack, 2},
		{"Ужин", allergenSafeMeals.dinner, 3},
	}
	for _, slot := range slots {
		if allergen := conflictsWithAllergens(menu.Meals[slot.index].Description, p.Allergies); allergen != "" {
			menu.Meals[slot.index] = slot.safe
			notes = append(notes, fmt.Sprintf("%s заменён: аллергия — %s", slot.name, allergen))
		}
	}
	return notes
}

// applyRestrictionsAndPhase применяет ограничения по здоровью, менструальную
// фазу и стресс к обеду/перекусу.
func applyRestrictionsAndPhase(menu *DayMenu, p DietProfile) []string {
	var notes []string

	snack, note := applyHealthRestrictions(p.Restrictions, menu.Meals[2])
	if note != "" {
		menu.Meals[2] = snack
		notes = append(notes, note)
	}
	if note := applyMenstrualNotes(p, &menu.Meals[1]); note != "" {
		notes = append(notes, note)
	}
	if note := applyStressNotes(p.StressLevel, &menu.Meals[2]); note != "" {
		notes = append(notes, note)
	}
	return notes
}

// scaleToCalories подгоняет калорийность под профиль пользователя.
func scaleToCalories(menu *DayMenu, p DietProfile) []string {
	target := estimateCalories(p)
	if target <= 0 {
		return nil
	}
	scale := float64(target) / float64(menu.TotalKcal)
	if scale <= 0.5 || scale >= 2.0 {
		return nil
	}
	*menu = scaleMenu(*menu, scale)
	return []string{fmt.Sprintf("Калорийность подстроена под профиль: ~%d ккал", target)}
}

// hasProfileExtras проверяет, есть ли что персонализировать.
func hasProfileExtras(p DietProfile) bool {
	return len(p.Allergies) > 0 || estimateCalories(p) > 0 ||
		p.MenstrualPhase != "" || p.StressLevel > 0 || len(p.Restrictions) > 0
}

// applyHealthRestrictions заменяет перекус при диабете и добавляет примечание.
func applyHealthRestrictions(restrictions []string, snack Meal) (Meal, string) {
	for _, raw := range restrictions {
		r := strings.ToLower(raw)
		if strings.Contains(r, "диабет") || strings.Contains(r, "diabetes") {
			return meal(mealSnack, "16:30", "Горсть орехов и яблоко", 10, 26, 14),
				"диабет: перекус без сахара"
		}
	}
	return snack, ""
}

// applyMenstrualNotes добавляет акцент на железо в фолликулярной фазе и
// менструации; в лютеиновую — на магний и сложные углеводы.
func applyMenstrualNotes(p DietProfile, lunch *Meal) string {
	if !isFemaleProfile(p.Gender) {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(p.MenstrualPhase)) {
	case "menstruation", "menstrual", "менструация":
		lunch.Description = lunch.Description + " (шпинат, чечевица — железо)"
		return "менструация: акцент на железо"
	case "luteal", "лютеиновая":
		return "лютеиновая фаза: больше магния и сложных углеводов"
	default:
		return ""
	}
}

// applyStressNotes снижает кофеин и добавляет магний.
func applyStressNotes(stress float64, snack *Meal) string {
	if stress < 7 {
		return ""
	}
	snack.Description = snack.Description + " + тёмный шоколад (магний)"
	return fmt.Sprintf("высокий стресс %.0f/10: меньше кофеина, больше магния", stress)
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
