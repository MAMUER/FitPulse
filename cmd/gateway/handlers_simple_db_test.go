package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleReqWithParam builds a request with a single chi URL param (e.g. meal_id).
func simpleReqWithParam(ctx context.Context, method, target, body, value string) *http.Request {
	r := catalogReq(ctx, method, target, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("meal_id", value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// These tests cover the DB-backed branches (g.db != nil) of handlers_simple.go,
// which the in-memory tests in handlers_simple_test.go do not exercise.

func TestSimpleDB_ListMeals_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT id, name, calories, meal_time").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "calories", "meal_time", "created_at"}).
				AddRow(int64(1), "Овсянка", 300, "08:00", time.Now()),
		)

	w := httptest.NewRecorder()
	g.listMealsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/nutrition/meals", ""))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Овсянка")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSimpleDB_ListMeals_QueryError(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("SELECT id, name, calories, meal_time").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.listMealsHandler(w, catalogReq(catalogCtx(), "GET", "/api/v1/nutrition/meals", ""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSimpleDB_CreateMeal_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("INSERT INTO nutrition_meals").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "name", "calories", "meal_time", "created_at"}).
				AddRow(int64(1), "Омлет", 250, "09:00", time.Now()),
		)

	w := httptest.NewRecorder()
	g.createMealHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/nutrition/meals",
		`{"name":"Омлет","calories":250,"time":"09:00"}`))

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Омлет")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSimpleDB_CreateMeal_Error(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectQuery("INSERT INTO nutrition_meals").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.createMealHandler(w, catalogReq(catalogCtx(), "POST", "/api/v1/nutrition/meals", `{"name":"x"}`))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSimpleDB_UpdateMeal_NotFound(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("UPDATE nutrition_meals").WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	g.updateMealHandler(w, simpleReqWithParam(catalogCtx(), "PUT", "/api/v1/nutrition/meals/1", `{"name":"x"}`, "1"))

	assert.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSimpleDB_UpdateMeal_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("UPDATE nutrition_meals").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.updateMealHandler(w, simpleReqWithParam(catalogCtx(), "PUT", "/api/v1/nutrition/meals/1", `{"name":"x"}`, "1"))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSimpleDB_UpdateMeal_Error(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("UPDATE nutrition_meals").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.updateMealHandler(w, simpleReqWithParam(catalogCtx(), "PUT", "/api/v1/nutrition/meals/1", `{"name":"x"}`, "1"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSimpleDB_DeleteMeal_InvalidID(t *testing.T) {
	db, _ := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	w := httptest.NewRecorder()
	g.deleteMealHandler(w, simpleReqWithParam(catalogCtx(), "DELETE", "/api/v1/nutrition/meals/x", "", "x"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSimpleDB_DeleteMeal_NotFound(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("DELETE FROM nutrition_meals").WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	g.deleteMealHandler(w, simpleReqWithParam(catalogCtx(), "DELETE", "/api/v1/nutrition/meals/1", "", "1"))

	assert.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSimpleDB_DeleteMeal_Success(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("DELETE FROM nutrition_meals").WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	g.deleteMealHandler(w, simpleReqWithParam(catalogCtx(), "DELETE", "/api/v1/nutrition/meals/1", "", "1"))

	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSimpleDB_DeleteMeal_Error(t *testing.T) {
	db, mock := newCatalogMockDB(t)
	g := newTestGateway()
	g.db = db

	mock.ExpectExec("DELETE FROM nutrition_meals").WillReturnError(errors.New("boom"))

	w := httptest.NewRecorder()
	g.deleteMealHandler(w, simpleReqWithParam(catalogCtx(), "DELETE", "/api/v1/nutrition/meals/1", "", "1"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
