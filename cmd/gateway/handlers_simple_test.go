package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
)

func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, userID)
}

func setupSimpleGateway() *gateway {
	log := &logger.Logger{Logger: zap.NewNop()}
	privateKeyPEM, publicKeyPEM, _ := jwt.GenerateES256KeyPair()
	return &gateway{
		log:           log,
		tokenProvider: jwt.NewJWTAdapter(privateKeyPEM, publicKeyPEM),
	}
}

func TestForgotPasswordHandler_MissingEmail(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/forgot-password", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.forgotPasswordHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPasswordHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/forgot-password", bytes.NewReader([]byte(`{"email":"test@example.com"}`)))
	req.Header.Set("Content-Type", "application/json")

	g.forgotPasswordHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestResetPasswordHandler_InvalidPayload(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/reset", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.resetPasswordHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPasswordHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/reset", bytes.NewReader([]byte(`{"email":"test@example.com","code":"123456","new_password":"newpass123"}`)))
	req.Header.Set("Content-Type", "application/json")

	g.resetPasswordHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestListVideosHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/videos", nil)

	g.listVideosHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "videos")
}

// ========== simpleStore tests ==========

func TestSimpleStore_AddMeal(t *testing.T) {
	s := newSimpleStore()
	m := s.addMeal("store-user-1", "Омлет", 300)

	assert.Equal(t, "meal-1", m.ID)
	assert.Equal(t, "Омлет", m.Name)
	assert.Equal(t, 300, m.Calories)
	assert.NotEmpty(t, m.Time)

	meals := s.getMeals("store-user-1")
	assert.Len(t, meals, 1)
	assert.Equal(t, "Омлет", meals[0].Name)
}

func TestSimpleStore_AddMeal_SeparateUsers(t *testing.T) {
	s := newSimpleStore()
	s.addMeal("store-user-1", "Завтрак", 200)
	s.addMeal("store-user-2", "Обед", 400)

	assert.Len(t, s.getMeals("store-user-1"), 1)
	assert.Len(t, s.getMeals("store-user-2"), 1)
	assert.Empty(t, s.getMeals("store-user-3"))
}

func TestSimpleStore_DeleteMeal(t *testing.T) {
	s := newSimpleStore()
	m := s.addMeal("store-user-1", "Омлет", 300)

	assert.True(t, s.deleteMeal("store-user-1", m.ID))
	assert.Empty(t, s.getMeals("store-user-1"))
	assert.False(t, s.deleteMeal("store-user-1", m.ID)) // already deleted
	assert.False(t, s.deleteMeal("store-user-1", "nonexistent"))
	assert.False(t, s.deleteMeal("store-user-2", m.ID)) // different user
}

func TestSimpleStore_AddEvent(t *testing.T) {
	s := newSimpleStore()
	e := s.addEvent("store-user-1", "Тренировка", "2024-01-15", "workout", "Утренняя")

	assert.Equal(t, "event-1", e.ID)
	assert.Equal(t, "Тренировка", e.Title)
	assert.Equal(t, "workout", e.Type)

	events := s.getEvents("store-user-1")
	assert.Len(t, events, 1)
	assert.Equal(t, "Тренировка", events[0].Title)
}

func TestSimpleStore_UpdateEvent(t *testing.T) {
	s := newSimpleStore()
	s.addEvent("store-user-1", "Бег", "2024-01-15", "cardio", "5км")

	updated := s.updateEvent("store-user-1", "event-1", "Бег вечерний", "2024-01-16", "running", "10км")
	assert.NotNil(t, updated)
	assert.Equal(t, "Бег вечерний", updated.Title)
	assert.Equal(t, "running", updated.Type)

	events := s.getEvents("store-user-1")
	assert.Len(t, events, 1)
	assert.Equal(t, "Бег вечерний", events[0].Title)
}

func TestSimpleStore_UpdateEvent_NotFound(t *testing.T) {
	s := newSimpleStore()
	s.addEvent("store-user-1", "Бег", "2024-01-15", "cardio", "5км")

	assert.Nil(t, s.updateEvent("store-user-1", "nonexistent", "X", "2024-01-16", "y", "z"))
	assert.Nil(t, s.updateEvent("store-user-2", "event-1", "X", "2024-01-16", "y", "z"))
}

func TestSimpleStore_DeleteEvent(t *testing.T) {
	s := newSimpleStore()
	s.addEvent("store-user-1", "Тренировка", "2024-01-15", "workout", "Утренняя")

	assert.True(t, s.deleteEvent("store-user-1", "event-1"))
	assert.Empty(t, s.getEvents("store-user-1"))
	assert.False(t, s.deleteEvent("store-user-1", "event-1")) // already deleted
	assert.False(t, s.deleteEvent("store-user-1", "nonexistent"))
}

// ========== Handler tests ==========

func TestListMealsHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/meals", nil)

	g.listMealsHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListMealsHandler_Success(t *testing.T) {
	g := setupSimpleGateway()
	globalStore.addMeal("list-meals-user", "Омлет", 300)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "list-meals-user"), "GET", "/api/v1/meals", nil)

	g.listMealsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Омлет")
	assert.Contains(t, w.Body.String(), "meals")
}

func TestCreateMealHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/meals", bytes.NewReader([]byte(`{"name":"Омлет","calories":300}`)))
	req.Header.Set("Content-Type", "application/json")

	g.createMealHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateMealHandler_BadRequest(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "create-meal-bad-user"), "POST", "/api/v1/meals", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.createMealHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateMealHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"name":"Омлет","calories":300}`)
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "create-meal-success-user"), "POST", "/api/v1/meals", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.createMealHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Омлет")
	assert.Contains(t, w.Body.String(), "300")
	assert.Contains(t, w.Body.String(), "meal")
}

func TestCreateMealHandler_Defaults(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{}`)
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "create-meal-defaults-user"), "POST", "/api/v1/meals", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.createMealHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Блюдо")
}

func TestDeleteMealHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("meal_id", "meal-1")
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/meals/meal-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteMealHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteMealHandler_MissingMealID(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	ctx := withUserID(context.Background(), "delete-meal-missing-user")
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/meals/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteMealHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteMealHandler_NotFound(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("meal_id", "nonexistent")
	ctx := withUserID(context.Background(), "delete-meal-notfound-user")
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/meals/nonexistent", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteMealHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteMealHandler_Success(t *testing.T) {
	g := setupSimpleGateway()
	m := globalStore.addMeal("delete-meal-success-user", "Омлет", 300)

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("meal_id", m.ID)
	ctx := withUserID(context.Background(), "delete-meal-success-user")
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/meals/"+m.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteMealHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	assert.Empty(t, globalStore.getMeals("delete-meal-success-user"))
}

func TestListCalendarEventsHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/events", nil)

	g.listCalendarEventsHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListCalendarEventsHandler_Success(t *testing.T) {
	g := setupSimpleGateway()
	globalStore.addEvent("list-events-user", "Тренировка", "2024-01-15", "workout", "Утренняя")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "list-events-user"), "GET", "/api/v1/events", nil)

	g.listCalendarEventsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Тренировка")
	assert.Contains(t, w.Body.String(), "events")
}

func TestCreateCalendarEventHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/events", bytes.NewReader([]byte(`{"title":"Тренировка"}`)))
	req.Header.Set("Content-Type", "application/json")

	g.createCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateCalendarEventHandler_BadRequest(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "create-event-bad-user"), "POST", "/api/v1/events", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.createCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCalendarEventHandler_Success(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"title":"Тренировка","date":"2024-01-15","type":"workout","description":"Утренняя"}`)
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "create-event-success-user"), "POST", "/api/v1/events", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.createCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Тренировка")
	assert.Contains(t, w.Body.String(), "event")
}

func TestCreateCalendarEventHandler_Defaults(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{}`)
	req := httptest.NewRequestWithContext(withUserID(context.Background(), "create-event-defaults-user"), "POST", "/api/v1/events", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.createCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Событие")
}

func TestUpdateCalendarEventHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("event_id", "event-1")
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, "PUT", "/api/v1/events/event-1", bytes.NewReader([]byte(`{}`)))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("Content-Type", "application/json")

	g.updateCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateCalendarEventHandler_MissingEventID(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	ctx := withUserID(context.Background(), "update-event-missing-user")
	req := httptest.NewRequestWithContext(ctx, "PUT", "/api/v1/events/", bytes.NewReader([]byte(`{}`)))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("Content-Type", "application/json")

	g.updateCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCalendarEventHandler_NotFound(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("event_id", "nonexistent")
	ctx := withUserID(context.Background(), "update-event-notfound-user")
	req := httptest.NewRequestWithContext(ctx, "PUT", "/api/v1/events/nonexistent", bytes.NewReader([]byte(`{}`)))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("Content-Type", "application/json")

	g.updateCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateCalendarEventHandler_Success(t *testing.T) {
	g := setupSimpleGateway()
	e := globalStore.addEvent("update-event-success-user", "Бег", "2024-01-15", "cardio", "5км")

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("event_id", e.ID)
	ctx := withUserID(context.Background(), "update-event-success-user")
	reqBody := []byte(`{"title":"Бег вечерний","date":"2024-01-16","type":"running","description":"10км"}`)
	req := httptest.NewRequestWithContext(ctx, "PUT", "/api/v1/events/"+e.ID, bytes.NewReader(reqBody))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("Content-Type", "application/json")

	g.updateCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Бег вечерний")
}

func TestDeleteCalendarEventHandler_Unauthorized(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("event_id", "event-1")
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/events/event-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteCalendarEventHandler_MissingEventID(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	ctx := withUserID(context.Background(), "delete-event-missing-user")
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/events/", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteCalendarEventHandler_NotFound(t *testing.T) {
	g := setupSimpleGateway()

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("event_id", "nonexistent")
	ctx := withUserID(context.Background(), "delete-event-notfound-user")
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/events/nonexistent", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteCalendarEventHandler_Success(t *testing.T) {
	g := setupSimpleGateway()
	e := globalStore.addEvent("delete-event-success-user", "Тренировка", "2024-01-15", "workout", "Утренняя")

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("event_id", e.ID)
	ctx := withUserID(context.Background(), "delete-event-success-user")
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/events/"+e.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.deleteCalendarEventHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	assert.Empty(t, globalStore.getEvents("delete-event-success-user"))
}
