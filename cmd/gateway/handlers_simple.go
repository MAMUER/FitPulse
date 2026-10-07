package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/MAMUER/project/internal/middleware"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type meal struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Calories  int       `json:"calories"`
	Time      string    `json:"time"`
	CreatedAt time.Time `json:"created_at"`
}

type calendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Date        string    `json:"date"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type video struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Minutes  int    `json:"minutes"`
	Kcal     int    `json:"kcal"`
	Type     string `json:"type"`
	VideoURL string `json:"video_url"`
}

type simpleStore struct {
	meals     map[string][]meal
	events    map[string][]calendarEvent
	mu        sync.RWMutex
	nextMeal  int
	nextEvent int
}

func newSimpleStore() *simpleStore {
	return &simpleStore{
		meals:  make(map[string][]meal),
		events: make(map[string][]calendarEvent),
	}
}

func (s *simpleStore) addMeal(userID, name string, calories int) meal {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextMeal++
	m := meal{
		ID:        "meal-" + string(rune(s.nextMeal)),
		Name:      name,
		Calories:  calories,
		Time:      time.Now().Format("15:04"),
		CreatedAt: time.Now(),
	}
	s.meals[userID] = append(s.meals[userID], m)
	return m
}

func (s *simpleStore) getMeals(userID string) []meal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]meal{}, s.meals[userID]...)
}

func (s *simpleStore) deleteMeal(userID, mealID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	meals, ok := s.meals[userID]
	if !ok {
		return false
	}
	for i, m := range meals {
		if m.ID == mealID {
			s.meals[userID] = append(meals[:i], meals[i+1:]...)
			return true
		}
	}
	return false
}

func (s *simpleStore) addEvent(userID, title, date, eventType, description string) calendarEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextEvent++
	e := calendarEvent{
		ID:          "event-" + string(rune(s.nextEvent)),
		Title:       title,
		Date:        date,
		Type:        eventType,
		Description: description,
		CreatedAt:   time.Now(),
	}
	s.events[userID] = append(s.events[userID], e)
	return e
}

func (s *simpleStore) getEvents(userID string) []calendarEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]calendarEvent{}, s.events[userID]...)
}

func (s *simpleStore) updateEvent(userID, eventID, title, date, eventType, description string) *calendarEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	events, ok := s.events[userID]
	if !ok {
		return nil
	}
	for i, e := range events {
		if e.ID == eventID {
			e.Title = title
			e.Date = date
			e.Type = eventType
			e.Description = description
			s.events[userID][i] = e
			return &e
		}
	}
	return nil
}

func (s *simpleStore) deleteEvent(userID, eventID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	events, ok := s.events[userID]
	if !ok {
		return false
	}
	for i, e := range events {
		if e.ID == eventID {
			s.events[userID] = append(events[:i], events[i+1:]...)
			return true
		}
	}
	return false
}

var globalStore = newSimpleStore()

func (g *gateway) listMealsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	meals := globalStore.getMeals(userID)
	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"meals":  meals,
	})
}

func (g *gateway) createMealHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	var req struct {
		Name     string `json:"name"`
		Calories int    `json:"calories"`
		Time     string `json:"time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "Блюдо"
	}
	if req.Calories <= 0 {
		req.Calories = 0
	}
	if req.Time == "" {
		req.Time = time.Now().Format("15:04")
	}
	meal := globalStore.addMeal(userID, req.Name, req.Calories)
	meal.Time = req.Time
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"meal":   meal,
	})
}

func (g *gateway) deleteMealHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	mealID := chi.URLParam(r, "meal_id")
	if mealID == "" {
		http.Error(w, "meal_id обязателен", http.StatusBadRequest)
		return
	}
	if !globalStore.deleteMeal(userID, mealID) {
		http.Error(w, "Блюдо не найдено", http.StatusNotFound)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
	})
}

func (g *gateway) listCalendarEventsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	events := globalStore.getEvents(userID)
	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"events": events,
	})
}

func (g *gateway) createCalendarEventHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	var req struct {
		Title       string `json:"title"`
		Date        string `json:"date"`
		Type        string `json:"type"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		req.Title = "Событие"
	}
	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}
	event := globalStore.addEvent(userID, req.Title, req.Date, req.Type, req.Description)
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"event":  event,
	})
}

func (g *gateway) updateCalendarEventHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	eventID := chi.URLParam(r, "event_id")
	if eventID == "" {
		http.Error(w, "event_id обязателен", http.StatusBadRequest)
		return
	}
	var req struct {
		Title       string `json:"title"`
		Date        string `json:"date"`
		Type        string `json:"type"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	event := globalStore.updateEvent(userID, eventID, req.Title, req.Date, req.Type, req.Description)
	if event == nil {
		http.Error(w, "Событие не найдено", http.StatusNotFound)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"event":  event,
	})
}

func (g *gateway) deleteCalendarEventHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	eventID := chi.URLParam(r, "event_id")
	if eventID == "" {
		http.Error(w, "event_id обязателен", http.StatusBadRequest)
		return
	}
	if !globalStore.deleteEvent(userID, eventID) {
		http.Error(w, "Событие не найдено", http.StatusNotFound)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
	})
}

func (g *gateway) listVideosHandler(w http.ResponseWriter, r *http.Request) {
	videos := []video{
		{ID: "v1", Title: "Кардио", Minutes: 20, Kcal: 180, Type: "cardio", VideoURL: "https://example.com/cardio.mp4"},
		{ID: "v2", Title: "Силовая", Minutes: 30, Kcal: 240, Type: "strength", VideoURL: "https://example.com/strength.mp4"},
		{ID: "v3", Title: "Йога", Minutes: 25, Kcal: 120, Type: "yoga", VideoURL: "https://example.com/yoga.mp4"},
		{ID: "v4", Title: "Растяжка", Minutes: 15, Kcal: 90, Type: "stretching", VideoURL: "https://example.com/stretching.mp4"},
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"videos": videos,
	})
}
