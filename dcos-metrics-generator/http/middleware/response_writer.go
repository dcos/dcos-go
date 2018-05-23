package middleware

import "net/http"

// statusRecorder is a simple wrapper over http.ResponseWriter
// which stores the response code in a variable `status` and calls the
// real implementation of ResponseWriter.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader wrapper function for http.ResponseWriter
func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}
