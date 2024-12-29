package awslambda

import (
	"bytes"
	"net/http"
)

type responseRecorder struct {
	Headers       http.Header
	Body          bytes.Buffer
	StatusCode    int
	writtenStatus bool
}

func (r *responseRecorder) Header() http.Header {
	return r.Headers
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.writtenStatus {
		r.WriteHeader(http.StatusOK)
	}
	return r.Body.Write(data)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.writtenStatus = true
}
