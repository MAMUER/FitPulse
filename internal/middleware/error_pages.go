package middleware

import (
	"bytes"
	"net/http"
	"os"
	"strings"
)

type errorPageRecorder struct {
	http.ResponseWriter
	headers    http.Header
	body       *bytes.Buffer
	statusCode int
	wrote      bool
}

func (r *errorPageRecorder) Header() http.Header {
	if r.headers == nil {
		r.headers = make(http.Header)
	}
	return r.headers
}

func (r *errorPageRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		if r.statusCode == 0 {
			r.statusCode = http.StatusOK
		}
		r.wrote = true
	}
	return r.body.Write(b)
}

func (r *errorPageRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.statusCode = code
	}
}

func (r *errorPageRecorder) replay(code int, body []byte) {
	if r.headers != nil {
		for k, v := range r.headers {
			r.ResponseWriter.Header()[k] = v
		}
	}
	r.ResponseWriter.WriteHeader(code)
	_, _ = r.ResponseWriter.Write(body)
}

const errorPageDir = "./web/static/errors"

func ErrorPages(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &errorPageRecorder{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
		}
		next.ServeHTTP(recorder, r)

		status := recorder.statusCode
		if status == 0 {
			status = http.StatusOK
		}

		accept := r.Header.Get("Accept")
		wantsHTML := strings.Contains(accept, "text/html")

		if wantsHTML && (status == http.StatusNotFound || status == http.StatusInternalServerError) {
			var file string
			if status == http.StatusNotFound {
				file = errorPageDir + "/error.html"
			} else {
				file = errorPageDir + "/error-500.html"
			}

			data, err := os.ReadFile(file)
			if err == nil {
				if recorder.headers != nil {
					for k, v := range recorder.headers {
						w.Header()[k] = v
					}
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.WriteHeader(status)
				_, _ = w.Write(data)
				return
			}
		}

		recorder.replay(status, recorder.body.Bytes())
	})
}
