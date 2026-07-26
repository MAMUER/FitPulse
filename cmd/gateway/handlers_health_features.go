package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

func (g *gateway) listHealthConditionsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
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
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) upsertHealthConditionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
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
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "condition": condition}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) deleteEntityHandler(w http.ResponseWriter, r *http.Request, paramName string, deleteFn func(string) error) {
	if _, ok := r.Context().Value(middleware.UserIDKey).(string); !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}
	entityID := chi.URLParam(r, paramName)
	if entityID == "" {
		http.Error(w, paramName+" требуется", http.StatusBadRequest)
		return
	}
	if err := deleteFn(entityID); err != nil {
		g.log.Error("Failed to delete entity", zap.String("param", paramName), zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

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

func (g *gateway) createBodyCompositionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
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
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "record": record}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) listBodyCompositionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
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
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) listMenstrualCyclesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
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
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) createMenstrualCycleHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
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
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "cycle": cycle}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) updateMenstrualCycleHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}
	cycleID := chi.URLParam(r, "cycle_id")
	if cycleID == "" {
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
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
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
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

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

func (g *gateway) syncDataHandler(w http.ResponseWriter, r *http.Request, syncFn func(string) (interface{}, error)) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}
	resp, err := syncFn(userID)
	if err != nil {
		g.log.Error("Failed to sync data", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "result": resp}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

type syncProviderRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (g *gateway) syncProviderHandler(w http.ResponseWriter, r *http.Request, syncFn func(context.Context, string, string, string) (interface{}, error)) {
	var req syncProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	syncWrapper := func(userID string) (interface{}, error) {
		return syncFn(r.Context(), userID, req.AccessToken, req.RefreshToken)
	}
	g.syncDataHandler(w, r, syncWrapper)
}

func (g *gateway) syncFloHandler(w http.ResponseWriter, r *http.Request) {
	g.syncProviderHandler(w, r, func(ctx context.Context, userID, accessToken, refreshToken string) (interface{}, error) {
		return g.userClient.SyncFloData(ctx, &userpb.SyncFloDataRequest{
			UserId: userID, AccessToken: accessToken, RefreshToken: refreshToken,
		})
	})
}

func (g *gateway) syncOKOKHandler(w http.ResponseWriter, r *http.Request) {
	g.syncProviderHandler(w, r, func(ctx context.Context, userID, accessToken, refreshToken string) (interface{}, error) {
		return g.userClient.SyncOKOKData(ctx, &userpb.SyncOKOKDataRequest{
			UserId: userID, AccessToken: accessToken, RefreshToken: refreshToken,
		})
	})
}
