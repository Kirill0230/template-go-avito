package response

import (
	"errors"
	"net/http"
	"testing"
)

type failingResponseWriter struct {
	header      http.Header
	writeErr    error
	recordedErr error
}

func (w *failingResponseWriter) Header() http.Header {
	return w.header
}

func (w *failingResponseWriter) WriteHeader(_ int) {}

func (w *failingResponseWriter) Write(_ []byte) (int, error) {
	return 0, w.writeErr
}

func (w *failingResponseWriter) SetError(err error) {
	w.recordedErr = err
}

func TestWriteJSONRecordsEncodingError(t *testing.T) {
	writeErr := errors.New("connection closed")
	writer := &failingResponseWriter{
		header:   make(http.Header),
		writeErr: writeErr,
	}

	WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})

	if !errors.Is(writer.recordedErr, writeErr) {
		t.Fatalf("recorded error = %v, want wrapped %v", writer.recordedErr, writeErr)
	}
}
