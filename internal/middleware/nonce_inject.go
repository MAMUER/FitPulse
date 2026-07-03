package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

func HTMLNonceInject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			next.ServeHTTP(w, r)
			return
		}

		nonce := GetNonce(r)
		if nonce == "" {
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(&nonceInjectWriter{
			ResponseWriter: w,
			nonce:          nonce,
		}, r)
	})
}

type nonceInjectWriter struct {
	http.ResponseWriter
	nonce     string
	committed bool
	buf       []byte
}

func (w *nonceInjectWriter) WriteHeader(status int) {
	if w.committed {
		return
	}
	w.committed = true
	if len(w.buf) > 0 {
		injected := strings.ReplaceAll(string(w.buf), `<script src=`, `<script nonce="`+w.nonce+`" src=`)
		injected = strings.ReplaceAll(injected, `<script\nsrc=`, `<script nonce="`+w.nonce+`"\nsrc=`)
		w.buf = nil
		w.ResponseWriter.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(status)
		_, _ = w.ResponseWriter.Write([]byte(injected))
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *nonceInjectWriter) Write(b []byte) (int, error) {
	if w.committed {
		n, err := w.ResponseWriter.Write(b)
		return n, fmt.Errorf("write response: %w", err)
	}
	w.buf = append(w.buf, b...)
	return len(b), nil
}
