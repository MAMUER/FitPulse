package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
)

// listDishesFromDB returns global + user dishes, optionally filtered by slot
// and/or a case-insensitive name substring.
func (g *gateway) listDishesFromDB(
	userID, slot, query string,
) ([]map[string]interface{}, error) {
	rows, err := g.db.Query(
		`SELECT id, name, meal_slot, kcal, protein_g, fat_g, carbs_g
		 FROM dishes
		 WHERE (owner_user_id IS NULL OR owner_user_id = $1)
		   AND ($2 = '' OR meal_slot = $2 OR meal_slot = 'any')
		   AND ($3 = '' OR name ILIKE '%' || $3 || '%')
		 ORDER BY owner_user_id NULLS FIRST, name`,
		userID, slot, query,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	dishes := []map[string]interface{}{}
	for rows.Next() {
		var (
			id            int64
			name, slot    string
			kcal, p, f, c int
		)
		if rows.Scan(&id, &name, &slot, &kcal, &p, &f, &c) != nil {
			continue
		}
		dishes = append(dishes, map[string]interface{}{
			"id": id, "name": name, "meal_slot": slot,
			"kcal": kcal, "protein_g": p, "fat_g": f, "carbs_g": c,
		})
	}
	return dishes, rows.Err()
}

// @Summary      List dishes
// @Description  Returns the dish catalog (global + user), optionally filtered.
// @Tags         dishes
// @Produce      json
// @Param        slot   query     string  false  "meal slot"
// @Param        q      query     string  false  "search query"
// @Success      200    {object}  map[string]interface{}
// @Router       /api/v1/dishes [get]
func (g *gateway) getDishesHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	if g.db == nil {
		writeJSON(w, map[string]interface{}{"status": "ok", "dishes": []interface{}{}})
		return
	}
	slot := r.URL.Query().Get("slot")
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	dishes, err := g.listDishesFromDB(userID, slot, query)
	if err != nil {
		g.log.Error("Failed to list dishes", zap.Error(err))
		http.Error(w, "failed to list dishes", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "dishes": dishes})
}

// @Summary      Add custom dish
// @Description  Adds a user-owned dish to the catalog.
// @Tags         dishes
// @Accept       json
// @Produce      json
// @Param        body  body      object  true  "Dish payload"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/v1/dishes [post]
func (g *gateway) addDishHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if g.db == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name     string `json:"name"`
		MealSlot string `json:"meal_slot"`
		Kcal     int    `json:"kcal"`
		ProteinG int    `json:"protein_g"`
		FatG     int    `json:"fat_g"`
		CarbsG   int    `json:"carbs_g"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	slot := strings.TrimSpace(req.MealSlot)
	if slot == "" {
		slot = "any"
	}

	var id int64
	err := g.db.QueryRow(
		`INSERT INTO dishes (owner_user_id, name, meal_slot, kcal, protein_g, fat_g, carbs_g)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		userID, name, slot, req.Kcal, req.ProteinG, req.FatG, req.CarbsG,
	).Scan(&id)
	if err != nil {
		g.log.Error("Failed to add dish", zap.Error(err))
		http.Error(w, "failed to add dish", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "id": id})
}
