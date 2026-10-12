package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func simpleRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()

	var reader io.Reader

	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	return httptest.NewRequestWithContext(withUserID(context.Background(), "simple-user-1"), method, target, reader)
}

func simpleParamRequest(t *testing.T, method, target, body, param, value string) *http.Request {
	t.Helper()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(param, value)

	req := simpleRequest(t, method, target, body)

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func simpleUnauthedRequest(method, target string) *http.Request {
	return httptest.NewRequest(method, target, nil)
}

func simpleDecodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	return payload
}

func TestSimpleStoreMeals(t *testing.T) {
	store := newSimpleStore()
	require.NotNil(t, store.meals)
	require.NotNil(t, store.events)
	assert.Empty(t, store.getMeals("simple-user-absent"))

	first := store.addMeal("simple-user-1", "Суп", 300)
	assert.Equal(t, "meal-1", first.ID)
	assert.Equal(t, "Суп", first.Name)
	assert.Equal(t, 300, first.Calories)
	assert.NotEmpty(t, first.Time)
	assert.False(t, first.CreatedAt.IsZero())

	second := store.addMeal("simple-user-1", "Салат", 150)
	assert.Equal(t, "meal-2", second.ID)
	assert.Len(t, store.getMeals("simple-user-1"), 2)

	assert.False(t, store.deleteMeal("simple-user-absent", first.ID))
	assert.False(t, store.deleteMeal("simple-user-1", "meal-missing"))

	assert.True(t, store.deleteMeal("simple-user-1", first.ID))
	assert.Len(t, store.getMeals("simple-user-1"), 1)
	assert.Equal(t, "meal-2", store.getMeals("simple-user-1")[0].ID)
}

func TestSimpleStoreEvents(t *testing.T) {
	store := newSimpleStore()
	assert.Empty(t, store.getEvents("simple-user-absent"))

	created := store.addEvent("simple-user-1", "Тренировка", "2024-05-01", "training", "Ноги")
	assert.Equal(t, "event-1", created.ID)
	assert.Equal(t, "2024-05-01", created.Date)

	assert.Len(t, store.getEvents("simple-user-1"), 1)

	assert.Nil(t, store.updateEvent("simple-user-absent", created.ID, "a", "b", "c", "d"))
	assert.Nil(t, store.updateEvent("simple-user-1", "event-missing", "a", "b", "c", "d"))

	updated := store.updateEvent("simple-user-1", created.ID, "Йога", "2024-05-02", "yoga", "Спина")
	require.NotNil(t, updated)
	assert.Equal(t, "Йога", updated.Title)
	assert.Equal(t, "2024-05-02", updated.Date)
	assert.Equal(t, "yoga", updated.Type)
	assert.Equal(t, "Спина", updated.Description)
	assert.Equal(t, "Йога", store.getEvents("simple-user-1")[0].Title)

	assert.False(t, store.deleteEvent("simple-user-absent", created.ID))
	assert.False(t, store.deleteEvent("simple-user-1", "event-missing"))

	assert.True(t, store.deleteEvent("simple-user-1", created.ID))
	assert.Empty(t, store.getEvents("simple-user-1"))
}

func TestSimpleListMealsHandler(t *testing.T) {
	g := newTestGateway()

	t.Run("unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.listMealsHandler(w, simpleUnauthedRequest("GET", "/api/v1/meals"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("lists user meals", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.listMealsHandler(w, simpleRequest(t, "GET", "/api/v1/meals", ""))

		assert.Equal(t, http.StatusOK, w.Code)

		payload := simpleDecodeBody(t, w)
		assert.Equal(t, "success", payload["status"])
		assert.NotNil(t, payload["meals"])
	})
}

func TestSimpleCreateMealHandler(t *testing.T) {
	g := newTestGateway()

	t.Run("unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createMealHandler(w, simpleUnauthedRequest("POST", "/api/v1/meals"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createMealHandler(w, simpleRequest(t, "POST", "/api/v1/meals", `{invalid`))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("applies defaults for empty fields", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createMealHandler(w, simpleRequest(t, "POST", "/api/v1/meals", `{"name":"","calories":-5,"time":""}`))

		assert.Equal(t, http.StatusCreated, w.Code)

		payload := simpleDecodeBody(t, w)
		meal, ok := payload["meal"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Блюдо", meal["name"])
		assert.Equal(t, float64(0), meal["calories"])
		assert.NotEmpty(t, meal["time"])
	})

	t.Run("keeps explicit values", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createMealHandler(w, simpleRequest(t, "POST", "/api/v1/meals", `{"name":"Каша","calories":420,"time":"08:30"}`))

		assert.Equal(t, http.StatusCreated, w.Code)

		payload := simpleDecodeBody(t, w)
		meal, ok := payload["meal"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Каша", meal["name"])
		assert.Equal(t, float64(420), meal["calories"])
		assert.Equal(t, "08:30", meal["time"])
	})
}

func TestSimpleDeleteMealHandler(t *testing.T) {
	g := newTestGateway()

	created := globalStore.addMeal("simple-user-1", "Суп", 100)

	t.Run("unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteMealHandler(w, simpleUnauthedRequest("DELETE", "/api/v1/meals/meal-1"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing meal id", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteMealHandler(w, simpleRequest(t, "DELETE", "/api/v1/meals/", ""))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("meal not found", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteMealHandler(w, simpleParamRequest(t, "DELETE", "/api/v1/meals/meal-missing", "", "meal_id", "meal-missing"))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("deletes meal", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteMealHandler(w, simpleParamRequest(t, "DELETE", "/api/v1/meals/meal-1", "", "meal_id", created.ID))

		assert.Equal(t, http.StatusOK, w.Code)

		payload := simpleDecodeBody(t, w)
		assert.Equal(t, "success", payload["status"])
	})
}

func TestSimpleCalendarEventHandlers(t *testing.T) {
	g := newTestGateway()

	created := globalStore.addEvent("simple-user-1", "Тренировка", "2024-05-01", "training", "Ноги")

	t.Run("list unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.listCalendarEventsHandler(w, simpleUnauthedRequest("GET", "/api/v1/calendar/events"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("list events", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.listCalendarEventsHandler(w, simpleRequest(t, "GET", "/api/v1/calendar/events", ""))

		assert.Equal(t, http.StatusOK, w.Code)

		payload := simpleDecodeBody(t, w)
		assert.Equal(t, "success", payload["status"])
		assert.NotNil(t, payload["events"])
	})

	t.Run("create unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createCalendarEventHandler(w, simpleUnauthedRequest("POST", "/api/v1/calendar/events"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("create invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createCalendarEventHandler(w, simpleRequest(t, "POST", "/api/v1/calendar/events", `{invalid`))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("create applies defaults", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createCalendarEventHandler(w, simpleRequest(t, "POST", "/api/v1/calendar/events", `{"title":"","date":""}`))

		assert.Equal(t, http.StatusCreated, w.Code)

		payload := simpleDecodeBody(t, w)
		event, ok := payload["event"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Событие", event["title"])
		assert.NotEmpty(t, event["date"])
	})

	t.Run("create keeps explicit values", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.createCalendarEventHandler(w, simpleRequest(t, "POST", "/api/v1/calendar/events", `{"title":"Бег","date":"2024-06-01","type":"cardio","description":"5 км"}`))

		assert.Equal(t, http.StatusCreated, w.Code)

		payload := simpleDecodeBody(t, w)
		event, ok := payload["event"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Бег", event["title"])
		assert.Equal(t, "2024-06-01", event["date"])
		assert.Equal(t, "cardio", event["type"])
		assert.Equal(t, "5 км", event["description"])
	})

	t.Run("update unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.updateCalendarEventHandler(w, simpleUnauthedRequest("PUT", "/api/v1/calendar/events/event-1"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("update missing event id", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.updateCalendarEventHandler(w, simpleRequest(t, "PUT", "/api/v1/calendar/events/", `{"title":"x"}`))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.updateCalendarEventHandler(w, simpleParamRequest(t, "PUT", "/api/v1/calendar/events/event-1", `{invalid`, "event_id", created.ID))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update event not found", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.updateCalendarEventHandler(w, simpleParamRequest(t, "PUT", "/api/v1/calendar/events/event-missing", `{"title":"x"}`, "event_id", "event-missing"))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("update event", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.updateCalendarEventHandler(w, simpleParamRequest(t, "PUT", "/api/v1/calendar/events/event-1", `{"title":"Плавание","date":"2024-05-02","type":"swimming","description":"1 км"}`, "event_id", created.ID))

		assert.Equal(t, http.StatusOK, w.Code)

		payload := simpleDecodeBody(t, w)
		event, ok := payload["event"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Плавание", event["title"])
		assert.Equal(t, "swimming", event["type"])
	})

	t.Run("delete unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteCalendarEventHandler(w, simpleUnauthedRequest("DELETE", "/api/v1/calendar/events/event-1"))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("delete missing event id", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteCalendarEventHandler(w, simpleRequest(t, "DELETE", "/api/v1/calendar/events/", ""))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete event not found", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteCalendarEventHandler(w, simpleParamRequest(t, "DELETE", "/api/v1/calendar/events/event-missing", "", "event_id", "event-missing"))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("delete event", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.deleteCalendarEventHandler(w, simpleParamRequest(t, "DELETE", "/api/v1/calendar/events/event-1", "", "event_id", created.ID))

		assert.Equal(t, http.StatusOK, w.Code)

		payload := simpleDecodeBody(t, w)
		assert.Equal(t, "success", payload["status"])
	})
}

func TestSimpleListVideosHandler(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()

	g.listVideosHandler(w, simpleRequest(t, "GET", "/api/v1/videos", ""))

	assert.Equal(t, http.StatusOK, w.Code)

	payload := simpleDecodeBody(t, w)
	assert.Equal(t, "success", payload["status"])

	videos, ok := payload["videos"].([]interface{})
	require.True(t, ok)
	assert.Len(t, videos, 4)
}
