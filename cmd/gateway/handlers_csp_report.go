package main

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// CSPReportBody описывает тело отчёта о нарушении CSP.
// Поддерживаются оба формата: старый (report-uri) и новый (report-to, Reporting API).
type CSPReportBody struct {
	// Формат Reporting API (report-to)
	Type      string           `json:"type"`
	URL       string           `json:"url"`
	UserAgent string           `json:"user_agent"`
	Body      CSPViolationBody `json:"body"`
	// Формат report-uri (устаревший, но всё ещё используется)
	Report CSPViolationBody `json:"csp-report"`
}

// CSPViolationBody содержит детали нарушения политики безопасности.
type CSPViolationBody struct {
	DocumentURI        string `json:"document-uri"`
	Referrer           string `json:"referrer"`
	BlockedURI         string `json:"blocked-uri"`
	ViolatedDirective  string `json:"violated-directive"`
	EffectiveDirective string `json:"effective-directive"`
	OriginalPolicy     string `json:"original-policy"`
	Disposition        string `json:"disposition"`
	SourceFile         string `json:"source-file"`
	LineNumber         int    `json:"line-number"`
	ColumnNumber       int    `json:"column-number"`
	StatusCode         int    `json:"status-code"`
	ScriptSample       string `json:"script-sample"`
}

// cspReportHandler принимает отчёты о нарушениях CSP от браузеров и
// перенаправляет их в лог-пайплайн (zap -> stdout -> ELK).
// Endpoint публичный: браузеры отправляют report анонимно, без заголовков авторизации.
func (g *gateway) cspReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var report CSPReportBody
	if err := json.Unmarshal(body, &report); err != nil {
		g.log.Warn("CSP report parse error", zap.Error(err), zap.ByteString("raw", body))
		w.WriteHeader(http.StatusOK)
		return
	}

	v := report.Body
	if report.Report.BlockedURI != "" || report.Report.ViolatedDirective != "" {
		v = report.Report
	}

	g.log.Warn("CSP_VIOLATION",
		zap.String("document_uri", v.DocumentURI),
		zap.String("blocked_uri", v.BlockedURI),
		zap.String("violated_directive", v.ViolatedDirective),
		zap.String("effective_directive", v.EffectiveDirective),
		zap.String("disposition", v.Disposition),
		zap.String("source_file", v.SourceFile),
		zap.Int("line_number", v.LineNumber),
		zap.Int("column_number", v.ColumnNumber),
		zap.String("script_sample", v.ScriptSample),
		zap.String("user_agent", r.Header.Get("User-Agent")),
		zap.String("client_ip", r.RemoteAddr),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `{}`)
}
