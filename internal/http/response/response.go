package response

import (
	"encoding/json"
	"net/http"
	"strings"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
)

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
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
	_ = json.NewEncoder(w).Encode(problem)
}

func WriteProblem(w http.ResponseWriter, r *http.Request, status int, problem api.Problem) {
	instance := r.URL.Path
	problem.Instance = &instance

	w.Header().Set(
		"Content-Type",
		"application/problem+json",
	)
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(problem)
}
