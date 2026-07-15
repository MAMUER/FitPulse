package main

import (
	"net/http"
)

// fitbitWebhookHandler временно возвращает 204 No Content.
// В будущем здесь будет логика верификации и обработки уведомлений от Fitbit.
func fitbitWebhookHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// withingsWebhookHandler временно возвращает 200 OK.
// В будущем здесь будет логика верификации подписи и обработки уведомлений от Withings.
func withingsWebhookHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
