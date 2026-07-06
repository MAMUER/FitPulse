package middleware

import (
	"fmt"
	"net/http"
	"regexp"
)

// scriptTagRE находит открывающие <script ...> теги, у которых ещё нет атрибута nonce.
var scriptTagRE = regexp.MustCompile(`(?i)<script\b([^>]*?)(?:\s|>)()`)

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
		injected := injectNonce(w.buf, w.nonce)
		w.buf = nil
		w.ResponseWriter.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(status)
		_, _ = w.ResponseWriter.Write([]byte(injected))
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

// injectNonce добавляет атрибут nonce во все <script> теги, у которых его ещё нет.
func injectNonce(body []byte, nonce string) string {
	return scriptTagRE.ReplaceAllStringFunc(string(body), func(tag string) string {
		if nonceAttrRE.MatchString(tag) {
			return tag
		}
		// <script ...> -> <script nonce="..." ...>
		return scriptTagRE.ReplaceAllString(tag, `<script nonce="`+nonce+`"$1>`)
	})
}

var nonceAttrRE = regexp.MustCompile(`(?i)\bnonce=`)

func (w *nonceInjectWriter) Write(b []byte) (int, error) {
	if w.committed {
		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, fmt.Errorf("write response: %w", err)
		}
		return n, nil
	}
	w.buf = append(w.buf, b...)
	return len(b), nil
}
