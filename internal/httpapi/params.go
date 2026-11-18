package httpapi

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, model.NewValidationError(name, "must be a valid UUID")
	}
	return id, nil
}

func queryInt(r *http.Request, name string) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, model.NewValidationError(name, "must be an integer")
	}
	return value, nil
}
