// Package chat provides rule-based FAQ and chat functionality.
package chat

import (
	"strings"
)

type FAQEntry struct {
	Keywords []string
	Answer   string
	Category string
}

var FAQDatabase = []FAQEntry{
	{
		Keywords: []string{"изменить", "поменять", "план"},
		Answer:   "Чтобы изменить план, перейдите в раздел Тренировки и нажмите 'Сгенерировать новый'.",
		Category: "plan",
	},
	{
		Keywords: []string{"начать", "старт", "запустить"},
		Answer:   "Для начала тренировок заполните анкету в разделе Профиль, затем нажмите 'Сгенерировать план'.",
		Category: "plan",
	},
	{
		Keywords: []string{"анкета", "опрос", "опросник"},
		Answer:   "Анкета помогает подобрать персональный план. Заполните её в разделе Профиль → Анкета.",
		Category: "survey",
	},
	{
		Keywords: []string{"уровень", "фитнес", "спорт"},
		Answer:   "Укажите свой уровень подготовки в анкете: beginner, intermediate или advanced.",
		Category: "profile",
	},
	{
		Keywords: []string{"цель", "цели", "хочу"},
		Answer:   "Выберите цель в анкете: похудение, набор мышц, выносливость или общая fitness.",
		Category: "profile",
	},
	{
		Keywords: []string{"боль", "болит", "травма", "противопоказания"},
		Answer:   "Укажите противопоказания в анкете, чтобы план был безопасным. При острых болях обратитесь к врачу.",
		Category: "health",
	},
	{
		Keywords: []string{"вес", "похудение", "сбросить", "похудеть"},
		Answer:   "Для похудения выбирайте кардио-тренировки и следите за питанием. Рекомендуется 3-4 занятия в неделю.",
		Category: "training",
	},
	{
		Keywords: []string{"мышцы", "масса", "сила", "набор"},
		Answer:   "Для набора мышечной массы выбирайте силовые тренировки с постепенным увеличением веса.",
		Category: "training",
	},
	{
		Keywords: []string{"выносливость", "длительность", "запас"},
		Answer:   "Для выносливости рекомендуются кардио-тренировки в аэробной зоне 45-90 минут.",
		Category: "training",
	},
	{
		Keywords: []string{"расписание", "дни", "когда"},
		Answer:   "Выберите удобные дни для тренировок при генерации плана. Рекомендуется 3-5 дней в неделю.",
		Category: "plan",
	},
	{
		Keywords: []string{"интенсивность", "сложно", "тяжело"},
		Answer:   "Интенсивность подбирается автоматически. Если слишком сложно — перегенерируйте план с другим уровнем.",
		Category: "training",
	},
	{
		Keywords: []string{"отдых", "восстановление", "сон"},
		Answer:   "Отдых — часть тренировочного процесса. Сон 7-9 часов и дни восстановления обязательны.",
		Category: "recovery",
	},
	{
		Keywords: []string{"питание", "еда", "диета", "калории"},
		Answer:   "Балансируйте белки, жиры и углеводы. Для точного расчёта калорий используйте раздел Питание.",
		Category: "nutrition",
	},
	{
		Keywords: []string{"вода", "пить", "гидратация"},
		Answer:   "Пейте 1.5-2 литра воды в день. Во время тренировки — по потребности.",
		Category: "nutrition",
	},
	{
		Keywords: []string{"сердечный", "пульс", "пульс"},
		Answer:   "Контролируйте пульс в аэробной зоне (60-80% от максимума) для безопасных тренировок.",
		Category: "health",
	},
	{
		Keywords: []string{"дома", "дом", "без зала"},
		Answer:   "Для тренировок дома выберите вес propioception и bodyweight упражнения в анкете.",
		Category: "training",
	},
	{
		Keywords: []string{"зал", "тренажёрный", "силовой"},
		Answer:   "В зале доступны все типы тренировок. Укажите 'gym' в анкете.",
		Category: "training",
	},
	{
		Keywords: []string{"удалить", "убрать", "отписаться"},
		Answer:   "Чтобы удалить аккаунт, обратитесь в поддержку через раздел Настройки.",
		Category: "account",
	},
	{
		Keywords: []string{"поддержка", "помощь", "вопрос"},
		Answer:   "Напишите нам на support@fitpulse.ru — мы ответим в течение 24 часов.",
		Category: "support",
	},
	{
		Keywords: []string{"отмена", "прервать", "остановить"},
		Answer:   "Чтобы остановить план — перейдите в Тренировки и нажмите 'Отменить'.",
		Category: "plan",
	},
	{
		Keywords: []string{"классификация", "состояние", "оценка"},
		Answer:   "Классификация оценивает ваше текущее состояние по биометрике и анкете для подбора плана.",
		Category: "ai",
	},
}

func FindAnswer(question string) string {
	lower := strings.ToLower(question)
	bestMatch := ""
	bestScore := 0

	for _, entry := range FAQDatabase {
		score := 0
		for _, kw := range entry.Keywords {
			if strings.Contains(lower, kw) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestMatch = entry.Answer
		}
	}

	if bestScore == 0 {
		return "Я не понял вопрос. Попробуйте перефразировать или выберите тему из списка."
	}
	return bestMatch
}
