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
