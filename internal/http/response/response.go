package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
)

type errorRecorder interface {
	SetError(error)
}

func RecordError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	if recorder, ok := w.(errorRecorder); ok {
		recorder.SetError(err)
	}
}

func encode(w http.ResponseWriter, value any) {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		RecordError(w, fmt.Errorf("encode response: %w", err))
	}
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	encode(w, value)
}

func WriteError(w http.ResponseWriter, r *http.Request, status int32, code, message string) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    http.StatusText(int(status)),
		Status:   status,
		Code:     code,
		Detail:   &message,
		Instance: &instance,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(status))
	encode(w, problem)
}

func WriteProblem(w http.ResponseWriter, r *http.Request, status int, problem api.Problem, cause error) {
	instance := r.URL.Path
	problem.Instance = &instance
	if status >= http.StatusInternalServerError {
		RecordError(w, cause)
	}

	w.Header().Set(
		"Content-Type",
		"application/problem+json",
	)
	w.WriteHeader(status)

	encode(w, problem)
}
