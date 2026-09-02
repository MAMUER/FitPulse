package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

// @Summary      List health conditions
// @Description  Retrieves a list of health conditions for the authenticated user
// @Tags         Health
// @Produce      json
// @Param        condition_type  query  string  false  "Filter by condition type"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/conditions [get]

func (g *gateway) listHealthConditionsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "listHealthConditions"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	conditionType := r.URL.Query().Get("condition_type")
	resp, err := g.userClient.ListHealthConditions(r.Context(), &userpb.ListHealthConditionsRequest{
		UserId: userID, ConditionType: conditionType,
	})
	if err != nil {
		g.log.Error("Failed to list health conditions", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	conditions := make([]map[string]interface{}, len(resp.Conditions))
	for i, c := range resp.Conditions {
		conditions[i] = map[string]interface{}{
			"id": c.Id, "user_id": c.UserId, "condition_type": c.ConditionType,
			"condition_name": c.ConditionName, "severity": c.Severity,
			"diagnosed_at": c.DiagnosedAt, "is_active": c.IsActive,
			"notes": c.Notes, "created_at": c.CreatedAt, "updated_at": c.UpdatedAt,
		}
	}
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "conditions": conditions, "total": resp.Total}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      Create or update health condition
// @Description  Creates or updates a health condition record for the authenticated user
// @Tags         Health
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Health condition data"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/conditions [post]

func (g *gateway) upsertHealthConditionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "upsertHealthCondition"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	var req struct {
		ConditionType string `json:"condition_type"`
		ConditionName string `json:"condition_name"`
		Severity      string `json:"severity"`
		DiagnosedAt   string `json:"diagnosed_at"`
		IsActive      bool   `json:"is_active"`
		Notes         string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode upsert health condition request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	condition, err := g.userClient.UpsertHealthCondition(r.Context(), &userpb.UpsertHealthConditionRequest{
		UserId: userID, ConditionType: req.ConditionType, ConditionName: req.ConditionName,
		Severity: req.Severity, DiagnosedAt: req.DiagnosedAt, IsActive: req.IsActive, Notes: req.Notes,
	})
	if err != nil {
		g.log.Error("Failed to upsert health condition", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "condition": condition}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

func (g *gateway) deleteEntityHandler(w http.ResponseWriter, r *http.Request, paramName string, deleteFn func(string) error) {
	if _, ok := r.Context().Value(middleware.UserIDKey).(string); !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "deleteEntity"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	entityID := chi.URLParam(r, paramName)
	if entityID == "" {
		g.log.Error("Missing entity ID in request", zap.String("param", paramName))
		http.Error(w, paramName+" требуется", http.StatusBadRequest)
		return
	}
	if err := deleteFn(entityID); err != nil {
		g.log.Error("Failed to delete entity", zap.String("param", paramName), zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      Delete health condition
// @Description  Deletes a specific health condition by ID
// @Tags         Health
// @Param        condition_id  path  string  true  "Health condition ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/conditions/{condition_id} [delete]

func (g *gateway) deleteHealthConditionHandler(w http.ResponseWriter, r *http.Request) {
	g.deleteEntityHandler(w, r, "condition_id", func(conditionID string) error {
		_, err := g.userClient.DeleteHealthCondition(r.Context(), &userpb.DeleteHealthConditionRequest{
			UserId: r.Context().Value(middleware.UserIDKey).(string), ConditionId: conditionID,
		})
		if err != nil {
			return fmt.Errorf("delete health condition: %w", err)
		}
		return nil
	})
}

// @Summary      Create body composition record
// @Description  Creates a new body composition record for the authenticated user
// @Tags         Health
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Body composition data"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/body-composition [post]

func (g *gateway) createBodyCompositionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "createBodyComposition"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	var req struct {
		RecordedAt           string  `json:"recorded_at"`
		WeightKg             float64 `json:"weight_kg"`
		HeightCm             int32   `json:"height_cm"`
		Bmi                  float64 `json:"bmi"`
		BodyFatPercentage    float64 `json:"body_fat_percentage"`
		MuscleMassPercentage float64 `json:"muscle_mass_percentage"`
		BoneMassPercentage   float64 `json:"bone_mass_percentage"`
		WaterPercentage      float64 `json:"water_percentage"`
		VisceralFatRating    int32   `json:"visceral_fat_rating"`
		MetabolicAge         int32   `json:"metabolic_age"`
		Source               string  `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode create body composition request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	record, err := g.userClient.CreateBodyComposition(r.Context(), &userpb.CreateBodyCompositionRequest{
		UserId: userID, RecordedAt: req.RecordedAt, WeightKg: req.WeightKg, HeightCm: req.HeightCm,
		Bmi: req.Bmi, BodyFatPercentage: req.BodyFatPercentage, MuscleMassPercentage: req.MuscleMassPercentage,
		BoneMassPercentage: req.BoneMassPercentage, WaterPercentage: req.WaterPercentage,
		VisceralFatRating: req.VisceralFatRating, MetabolicAge: req.MetabolicAge, Source: req.Source,
	})
	if err != nil {
		g.log.Error("Failed to create body composition record", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "record": record}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      List body composition records
// @Description  Retrieves body composition records for the authenticated user with optional date range filtering
// @Tags         Health
// @Produce      json
// @Param        from   query  string  false  "Start date filter (RFC3339)"
// @Param        to     query  string  false  "End date filter (RFC3339)"
// @Param        limit  query  int     false  "Maximum records (default 100, max 10000)"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/body-composition [get]

func (g *gateway) listBodyCompositionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "listBodyComposition"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	limit := 100
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 10000 {
		limit = l
	}
	resp, err := g.userClient.ListBodyComposition(r.Context(), &userpb.ListBodyCompositionRequest{
		UserId: userID, From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"), Limit: int32(limit),
	})
	if err != nil {
		g.log.Error("Failed to list body composition", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	records := make([]map[string]interface{}, len(resp.Records))
	for i, rec := range resp.Records {
		records[i] = map[string]interface{}{
			"id": rec.Id, "user_id": rec.UserId, "recorded_at": rec.RecordedAt,
			"weight_kg": rec.WeightKg, "height_cm": rec.HeightCm, "bmi": rec.Bmi,
			"body_fat_percentage": rec.BodyFatPercentage, "muscle_mass_percentage": rec.MuscleMassPercentage,
			"bone_mass_percentage": rec.BoneMassPercentage, "water_percentage": rec.WaterPercentage,
			"visceral_fat_rating": rec.VisceralFatRating, "metabolic_age": rec.MetabolicAge,
			"source": rec.Source, "created_at": rec.CreatedAt,
		}
	}
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "records": records, "total": resp.Total}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      List menstrual cycles
// @Description  Retrieves menstrual cycle records for the authenticated user
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/menstrual-cycles [get]

func (g *gateway) listMenstrualCyclesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "listMenstrualCycles"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	resp, err := g.userClient.ListMenstrualCycles(r.Context(), &userpb.ListMenstrualCyclesRequest{UserId: userID})
	if err != nil {
		g.log.Error("Failed to list menstrual cycles", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "cycles": resp.Cycles, "total": resp.Total}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      Create menstrual cycle record
// @Description  Creates a new menstrual cycle record for the authenticated user
// @Tags         Health
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Menstrual cycle data"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/menstrual-cycles [post]

func (g *gateway) createMenstrualCycleHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "createMenstrualCycle"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	var req struct {
		CycleStartDate string   `json:"cycle_start_date"`
		CycleEndDate   string   `json:"cycle_end_date"`
		FlowIntensity  string   `json:"flow_intensity"`
		Symptoms       []string `json:"symptoms"`
		Moods          []string `json:"moods"`
		Notes          string   `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode create menstrual cycle request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	cycle, err := g.userClient.CreateMenstrualCycle(r.Context(), &userpb.CreateMenstrualCycleRequest{
		UserId: userID, CycleStartDate: req.CycleStartDate, CycleEndDate: req.CycleEndDate,
		FlowIntensity: req.FlowIntensity, Symptoms: req.Symptoms, Moods: req.Moods, Notes: req.Notes,
	})
	if err != nil {
		g.log.Error("Failed to create menstrual cycle", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "cycle": cycle}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      Update menstrual cycle record
// @Description  Updates an existing menstrual cycle record by ID
// @Tags         Health
// @Accept       json
// @Produce      json
// @Param        cycle_id  path  string  true  "Menstrual cycle ID"
// @Param        request   body  object  required  "Updated menstrual cycle data"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/menstrual-cycles/{cycle_id} [put]

func (g *gateway) updateMenstrualCycleHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "updateMenstrualCycle"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	cycleID := chi.URLParam(r, "cycle_id")
	if cycleID == "" {
		g.log.Error("Missing cycle_id in request")
		http.Error(w, "cycle_id требуется", http.StatusBadRequest)
		return
	}
	var req struct {
		CycleStartDate string   `json:"cycle_start_date"`
		CycleEndDate   string   `json:"cycle_end_date"`
		FlowIntensity  string   `json:"flow_intensity"`
		Symptoms       []string `json:"symptoms"`
		Moods          []string `json:"moods"`
		Notes          string   `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode update menstrual cycle request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	cycle, err := g.userClient.UpdateMenstrualCycle(r.Context(), &userpb.UpdateMenstrualCycleRequest{
		UserId: userID, CycleId: cycleID, CycleStartDate: req.CycleStartDate,
		CycleEndDate: req.CycleEndDate, FlowIntensity: req.FlowIntensity,
		Symptoms: req.Symptoms, Moods: req.Moods, Notes: req.Notes,
	})
	if err != nil {
		g.log.Error("Failed to update menstrual cycle", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "cycle": cycle}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      Delete menstrual cycle record
// @Description  Deletes a specific menstrual cycle record by ID
// @Tags         Health
// @Param        cycle_id  path  string  true  "Menstrual cycle ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/health/menstrual-cycles/{cycle_id} [delete]

func (g *gateway) deleteMenstrualCycleHandler(w http.ResponseWriter, r *http.Request) {
	g.deleteEntityHandler(w, r, "cycle_id", func(cycleID string) error {
		_, err := g.userClient.DeleteMenstrualCycle(r.Context(), &userpb.DeleteMenstrualCycleRequest{
			UserId: r.Context().Value(middleware.UserIDKey).(string), CycleId: cycleID,
		})
		if err != nil {
			return fmt.Errorf("delete menstrual cycle: %w", err)
		}
		return nil
	})
}
