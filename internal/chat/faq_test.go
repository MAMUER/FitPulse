package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindAnswer_ExactMatch(t *testing.T) {
	answer := FindAnswer("Как изменить план?")
	assert.Equal(t, "Чтобы изменить план, перейдите в раздел Тренировки и нажмите 'Сгенерировать новый'.", answer)
}

func TestFindAnswer_PartialMatch(t *testing.T) {
	answer := FindAnswer("хочу похудеть")
	assert.Contains(t, answer, "похудение")
}

func TestFindAnswer_NoMatch(t *testing.T) {
	answer := FindAnswer("xyz123")
	assert.Equal(t, "Я не понял вопрос. Попробуйте перефразировать или выберите тему из списка.", answer)
}

func TestFindAnswer_CaseInsensitive(t *testing.T) {
	answer := FindAnswer("ПОМЕНЯТЬ ПЛАН")
	assert.Equal(t, "Чтобы изменить план, перейдите в раздел Тренировки и нажмите 'Сгенерировать новый'.", answer)
}

func TestFindAnswer_MultipleKeywords(t *testing.T) {
	answer := FindAnswer("как начать тренировки")
	assert.Contains(t, answer, "анкету")
}

func TestFindAnswer_EmptyQuestion(t *testing.T) {
	answer := FindAnswer("")
	assert.Equal(t, "Я не понял вопрос. Попробуйте перефразировать или выберите тему из списка.", answer)
}

func TestLoadFAQFromJSON(t *testing.T) {
	data := []byte(`[
		{
			"id": "faq_00001",
			"intent": "exercise_howto",
			"question": "Как делать 3/4 Sit-Up?",
			"answer": "3/4 Sit-Up. Целевые мышцы: abdominals. Оборудование: body only.",
			"context": {
				"exercise": "3/4 Sit-Up",
				"source": "exercises_json",
				"level": "beginner",
				"category": "strength",
				"equipment": "body only",
				"primary_muscles": ["abdominals"],
				"secondary_muscles": [],
				"target_muscles": [],
				"body_parts": []
			}
		}
	]`)

	entries, err := LoadFAQFromJSON(data)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "faq_00001", entries[0].ID)
	assert.Equal(t, "Как делать 3/4 Sit-Up?", entries[0].Question)
	assert.Equal(t, "3/4 Sit-Up. Целевые мышцы: abdominals. Оборудование: body only.", entries[0].Answer)
	assert.Equal(t, "beginner", entries[0].Context.Level)
	assert.Equal(t, "strength", entries[0].Context.Category)
	assert.Equal(t, "abdominals", entries[0].Context.PrimaryMuscles[0])
	assert.Equal(t, "body only", entries[0].Context.Equipment)
}

func TestLoadFAQFromJSON_InvalidJSON(t *testing.T) {
	_, err := LoadFAQFromJSON([]byte(`invalid`))
	assert.Error(t, err)
}
