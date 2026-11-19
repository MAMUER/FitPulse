package chat

import (
	"fmt"
	"strings"
)

// Meal is one eating occasion in a generated day menu.
type Meal struct {
	Name        string `json:"name"`
	Time        string `json:"time"`
	ProteinG    int    `json:"protein_g"`
	CarbsG      int    `json:"carbs_g"`
	FatG        int    `json:"fat_g"`
	Kcal        int    `json:"kcal"`
	Description string `json:"description"`
}

// DayMenu is a deterministic, generated nutrition plan for one day. It is
// produced from the user's request without any external LLM: the goal is
// inferred from keywords and the meals are picked from fixed per-goal
// templates, so the same request always yields the same menu.
type DayMenu struct {
	Goal      string `json:"goal"`
	GoalLabel string `json:"goal_label"`
	TotalKcal int    `json:"total_kcal"`
	ProteinG  int    `json:"protein_g"`
	CarbsG    int    `json:"carbs_g"`
	FatG      int    `json:"fat_g"`
	WaterMl   int    `json:"water_ml"`
	Meals     []Meal `json:"meals"`
	Note      string `json:"note"`
}

// kcal computes the energy value of a meal from its macros.
func kcal(protein, carbs, fat int) int {
	return protein*4 + carbs*4 + fat*9
}

func meal(name, time, description string, protein, carbs, fat int) Meal {
	return Meal{
		Name:        name,
		Time:        time,
		ProteinG:    protein,
		CarbsG:      carbs,
		FatG:        fat,
		Kcal:        kcal(protein, carbs, fat),
		Description: description,
	}
}

// menuTemplate is a full day of meals for one goal.
type menuTemplate struct {
	label     string
	waterMl   int
	note      string
	breakfast Meal
	lunch     Meal
	snack     Meal
	dinner    Meal
}

var menuTemplates = map[string]menuTemplate{
	"weight_loss": {
		label:     "Снижение веса",
		waterMl:   2000,
		note:      "Лёгкий дефицит калорий, упор на белок и клетчатку. Не является медицинским советом — при заболеваниях или травмах проконсультируйтесь с врачом.",
		breakfast: meal("Завтрак", "08:00", "Овсянка на воде с ягодами и горстью орехов", 15, 40, 10),
		lunch:     meal("Обед", "13:00", "Куриная грудка-гриль с гречкой и тушёными овощами", 40, 35, 12),
		snack:     meal("Перекус", "16:30", "Греческий йогурт с яблоком", 10, 18, 3),
		dinner:    meal("Ужин", "19:30", "Запечённый лосось с брокколи", 35, 12, 18),
	},
	"muscle_gain": {
		label:     "Набор мышечной массы",
		waterMl:   2500,
		note:      "Профицит калорий с высоким содержанием белка для роста мышц. Не является медицинским советом — при заболеваниях или травмах проконсультируйтесь с врачом.",
		breakfast: meal("Завтрак", "08:00", "Омлет из 3 яиц с тостом из цельнозернового хлеба и авокадо", 30, 60, 18),
		lunch:     meal("Обед", "13:00", "Говядина с рисом и овощами", 50, 70, 20),
		snack:     meal("Перекус", "16:30", "Творог с бананом и мёдом", 25, 40, 10),
		dinner:    meal("Ужин", "19:30", "Индейка с пастой из твёрдых сортов и салатом", 45, 55, 18),
	},
	"balance": {
		label:     "Сбалансированное питание",
		waterMl:   2200,
		note:      "Сбалансированный рацион без крайностей. Не является медицинским советом — при заболеваниях или травмах проконсультируйтесь с врачом.",
		breakfast: meal("Завтрак", "08:00", "Каша с фруктами, орехами и ложкой мёда", 20, 50, 14),
		lunch:     meal("Обед", "13:00", "Булгур с курицей, запечёнными овощами и зеленью", 40, 50, 16),
		snack:     meal("Перекус", "16:30", "Яблоко и горсть миндаля", 12, 25, 6),
		dinner:    meal("Ужин", "19:30", "Треска с киноа и салатом из свежих овощей", 38, 30, 18),
	},
	// "light_dinner" — обычный день, но ужин заменён на лёгкий (кнопка «Ужин»).
	"light_dinner": {
		label:     "Лёгкий ужин",
		waterMl:   2200,
		note:      "Дневной рацион с заменой ужина на лёгкий вариант. Не является медицинским советом — при заболеваниях или травмах проконсультируйтесь с врачом.",
		breakfast: meal("Завтрак", "08:00", "Каша с фруктами, орехами и ложкой мёда", 20, 50, 14),
		lunch:     meal("Обед", "13:00", "Булгур с курицей, запечёнными овощами и зеленью", 40, 55, 16),
		snack:     meal("Перекус", "16:30", "Яблоко и горсть миндаля", 12, 20, 5),
		dinner:    meal("Ужин (лёгкий)", "19:00", "Овощной салат с тунцом и оливковым маслом", 25, 15, 8),
	},
}

// IsMenuRequest reports whether the question asks to compose/plan a menu or
// meals (as opposed to a general FAQ question).
func IsMenuRequest(question string) bool {
	q := strings.ToLower(question)
	menuWords := []string{
		"меню", "рацион", "питание", "план питания", "составь меню",
		"завтрак", "обед", "ужин", "перекус", "еда", "еду", "поесть",
		"meal", "menu", "diet", "breakfast", "lunch", "dinner",
	}
	for _, w := range menuWords {
		if strings.Contains(q, w) {
			return true
		}
	}
	return false
}

// DetectGoal infers a nutrition goal key from the question text.
func DetectGoal(question string) string {
	q := strings.ToLower(question)
	switch {
	case strings.Contains(q, "ужен") || strings.Contains(q, "легк") ||
		strings.Contains(q, "лёг") || strings.Contains(q, "dinner") ||
		strings.Contains(q, "light"):
		return "light_dinner"
	case strings.Contains(q, "похуд") || strings.Contains(q, "сброс") ||
		strings.Contains(q, "сбросить") || strings.Contains(q, "вес") ||
		strings.Contains(q, "дефиц") || strings.Contains(q, "cut") ||
		strings.Contains(q, "weight"):
		return "weight_loss"
	case strings.Contains(q, "мышц") || strings.Contains(q, "масс") ||
		strings.Contains(q, "набор") || strings.Contains(q, "протеин") ||
		strings.Contains(q, "белок") || strings.Contains(q, "bulk") ||
		strings.Contains(q, "muscle"):
		return "muscle_gain"
	default:
		return "balance"
	}
}

// BuildDayMenu builds a deterministic day menu for the given goal key.
func BuildDayMenu(goal string) DayMenu {
	tmpl, ok := menuTemplates[goal]
	if !ok {
		tmpl = menuTemplates["balance"]
		goal = "balance"
	}
	meals := []Meal{tmpl.breakfast, tmpl.lunch, tmpl.snack, tmpl.dinner}

	var totalKcal, totalP, totalC, totalF int
	for _, m := range meals {
		totalKcal += m.Kcal
		totalP += m.ProteinG
		totalC += m.CarbsG
		totalF += m.FatG
	}

	return DayMenu{
		Goal:      goal,
		GoalLabel: tmpl.label,
		TotalKcal: totalKcal,
		ProteinG:  totalP,
		CarbsG:    totalC,
		FatG:      totalF,
		WaterMl:   tmpl.waterMl,
		Meals:     meals,
		Note:      tmpl.note,
	}
}

// MenuSummary builds a human-readable summary of a generated menu.
func MenuSummary(menu DayMenu) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Составил меню на сегодня (%s, ~%d ккал).", menu.GoalLabel, menu.TotalKcal)
	for _, m := range menu.Meals {
		fmt.Fprintf(
			&b, "\n• %s (%s): %s — %d ккал (Б%d Ж%d У%d).",
			m.Name, m.Time, m.Description, m.Kcal,
			m.ProteinG, m.FatG, m.CarbsG,
		)
	}
	fmt.Fprintf(&b, "\nВода: ~%.1f л. %s", float64(menu.WaterMl)/1000.0, menu.Note)
	return b.String()
}

// Respond produces a chat answer for a question. When the question asks for a
// menu, it returns a generated DayMenu (and a summary answer); otherwise it
// falls back to the rule-based FAQ.
func Respond(question string) (answer string, menu *DayMenu) {
	if IsMenuRequest(question) {
		m := BuildDayMenu(DetectGoal(question))
		return MenuSummary(m), &m
	}
	return FindAnswer(question), nil
}
