// File: internal/server/middleware.go
package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// metricsMiddleware records HTTP request metrics
func (s *HTTPServer) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(wrapper, r)

		// Record metrics
		duration := time.Since(start)
		status := strconv.Itoa(wrapper.statusCode)
		path := s.getRoutePath(r)

		if s.metricsManager != nil {
			s.metricsManager.GetPrometheusMetrics().RecordHTTPRequest(
				r.Method,
				path,
				status,
				duration,
			)
		}
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// getRoutePath extracts the route template from the request
func (s *HTTPServer) getRoutePath(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return r.URL.Path
	}

	template, err := route.GetPathTemplate()
	if err != nil {
		return r.URL.Path
	}

	return template
}
