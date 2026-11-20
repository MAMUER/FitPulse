package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
)

// @Summary      Report a user
// @Description  Stores a report (e.g. about an inappropriate photo) and
// @Description  forwards it to the oncall-bot (Telegram) when configured.
// @Tags         reports
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/report [post]
func (g *gateway) reportHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	var req struct {
		ReportedUserID string `json:"reported_user_id"`
		Reason         string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, msgInvalidRequest, http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Жалоба на фото"
	}
	reportedID := strings.TrimSpace(req.ReportedUserID)

	if g.db != nil {
		if _, err := g.db.Exec(
			`INSERT INTO reports (reporter_id, reported_user_id, reason)
			 VALUES ($1, $2, $3)`,
			userID, reportedID, reason,
		); err != nil {
			g.log.Error("Failed to store report", zap.Error(err))
		}
	}

	g.notifyOncall(userID, reportedID, reason)
	writeJSON(w, map[string]interface{}{"status": "ok"})
}

// getEnvOrFile returns the env value, or the contents of the file referenced by
// <NAME>_FILE (used for secrets mounted from k8s).
func getEnvOrFile(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	if path := os.Getenv(name + "_FILE"); path != "" {
		//nolint:gosec // G304: путь берётся из доверенного env (файл из k8s-секрета).
		if b, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// notifyOncall sends a short message to the oncall-bot via the Telegram Bot API
// when TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are configured.
func (g *gateway) notifyOncall(reporterID, reportedID, reason string) {
	token := getEnvOrFile("TELEGRAM_BOT_TOKEN")
	chatID := getEnvOrFile("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return
	}

	text := "🚨 Жалоба: reporter=" + reporterID + " reported=" + reportedID + " reason=" + reason
	form := url.Values{"chat_id": {chatID}, "text": {text}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//nolint:gosec // G704: хост фиксирован (api.telegram.org), токен берётся из env.
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		"https://api.telegram.org/bot"+token+"/sendMessage",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		g.log.Warn("Failed to build oncall request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	//nolint:gosec // G704: фиксированный хост, адрес из доверенного env.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		g.log.Warn("Failed to send oncall notification", zap.Error(err))
		return
	}
	_ = resp.Body.Close()
}
