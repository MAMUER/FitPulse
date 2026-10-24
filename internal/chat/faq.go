// Package chat provides rule-based FAQ and chat functionality.
package chat

import (
	"encoding/json"
	"fmt"
	"strings"
)

type FAQContext struct {
	Exercise         string   `json:"exercise"`
	Source           string   `json:"source"`
	Level            string   `json:"level"`
	Category         string   `json:"category"`
	Equipment        string   `json:"equipment"`
	PrimaryMuscles   []string `json:"primary_muscles"`
	SecondaryMuscles []string `json:"secondary_muscles"`
	TargetMuscles    []string `json:"target_muscles"`
	BodyParts        []string `json:"body_parts"`
}

type FAQEntry struct {
	ID       string     `json:"id"`
	Intent   string     `json:"intent"`
	Question string     `json:"question"`
	Answer   string     `json:"answer"`
	Context  FAQContext `json:"context"`
	Category string
	Level    string
	Keywords []string
}

func faq(keywords []string, answer, category string) FAQEntry {
	return FAQEntry{
		Keywords: keywords,
		Answer:   answer,
		Category: category,
	}
}

var FAQDatabase = []FAQEntry{
	faq([]string{"изменить", "поменять", "план"}, "Чтобы изменить план, перейдите в раздел Тренировки и нажмите 'Сгенерировать новый'.", "plan"),
	faq([]string{"начать", "старт", "запустить"}, "Для начала тренировок заполните анкету в разделе Профиль, затем нажмите 'Сгенерировать план'.", "plan"),
	faq([]string{"анкета", "опрос", "опросник"}, "Анкета помогает подобрать персональный план. Заполните её в разделе Профиль → Анкета.", "survey"),
	faq([]string{"уровень", "фитнес", "спорт"}, "Укажите свой уровень подготовки в анкете: beginner, intermediate или advanced.", "profile"),
	faq([]string{"цель", "цели", "хочу"}, "Выберите цель в анкете: похудение, набор мышц, выносливость или общая fitness.", "profile"),
	faq([]string{"боль", "болит", "травма", "противопоказания"}, "Укажите противопоказания в анкете, чтобы план был безопасным. При острых болях обратитесь к врачу.", "health"),
	faq([]string{"вес", "похудение", "сбросить", "похудеть"}, "Для похудения выбирайте кардио-тренировки и следите за питанием. Рекомендуется 3-4 занятия в неделю.", "training"),
	faq([]string{"мышцы", "масса", "сила", "набор"}, "Для набора мышечной массы выбирайте силовые тренировки с постепенным увеличением веса.", "training"),
	faq([]string{"выносливость", "длительность", "запас"}, "Для выносливости рекомендуются кардио-тренировки в аэробной зоне 45-90 минут.", "training"),
	faq([]string{"расписание", "дни", "когда"}, "Выберите удобные дни для тренировок при генерации плана. Рекомендуется 3-5 дней в неделю.", "plan"),
	faq([]string{"интенсивность", "сложно", "тяжело"}, "Интенсивность подбирается автоматически. Если слишком сложно — перегенерируйте план с другим уровнем.", "training"),
	faq([]string{"отдых", "восстановление", "сон"}, "Отдых — часть тренировочного процесса. Сон 7-9 часов и дни восстановления обязательны.", "recovery"),
	faq([]string{"питание", "еда", "диета", "калории"}, "Балансируйте белки, жиры и углеводы. Для точного расчёта калорий используйте раздел Питание.", "nutrition"),
	faq([]string{"вода", "пить", "гидратация"}, "Пейте 1.5-2 литра воды в день. Во время тренировки — по потребности.", "nutrition"),
	faq([]string{"сердечный", "пульс", "пульс"}, "Контролируйте пульс в аэробной зоне (60-80% от максимума) для безопасных тренировок.", "health"),
	faq([]string{"дома", "дом", "без зала"}, "Для тренировок дома выберите bodyweight упражнения в анкете.", "training"),
	faq([]string{"зал", "тренажёрный", "силовой"}, "В зале доступны все типы тренировок. Укажите 'gym' в анкете.", "training"),
	faq([]string{"удалить", "убрать", "отписаться"}, "Чтобы удалить аккаунт, обратитесь в поддержку через раздел Настройки.", "account"),
	faq([]string{"поддержка", "помощь", "вопрос"}, "Напишите нам на support@fitpulse.ru — мы ответим в течение 24 часов.", "support"),
	faq([]string{"отмена", "прервать", "остановить"}, "Чтобы остановить план — перейдите в Тренировки и нажмите 'Отменить'.", "plan"),
	faq([]string{"классификация", "состояние", "оценка"}, "Классификация оценивает ваше текущее состояние по биометрике и анкете для подбора плана.", "ai"),
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

func LoadFAQFromJSON(data []byte) ([]FAQEntry, error) {
	var entries []FAQEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FAQ JSON: %w", err)
	}
	return entries, nil
}
