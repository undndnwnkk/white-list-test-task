package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

const maxBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return describeDecodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return model.NewValidationError("body", "must contain a single JSON object")
	}
	return nil
}

func describeDecodeError(err error) error {
	var (
		syntaxErr   *json.SyntaxError
		typeErr     *json.UnmarshalTypeError
		timeErr     *time.ParseError
		maxBytesErr *http.MaxBytesError
	)

	switch {
	case errors.Is(err, io.EOF):
		return model.NewValidationError("body", "must not be empty")
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return model.NewValidationError("body", "malformed JSON")
	case errors.As(err, &typeErr):
		return model.NewValidationError(typeErr.Field, fmt.Sprintf("must be of type %s", typeErr.Type))
	case errors.As(err, &timeErr):
		return model.NewValidationError("body", "time must be in ISO 8601 format, e.g. 2030-01-07T05:00:00Z")
	case errors.As(err, &maxBytesErr):
		return model.NewValidationError("body", "is too large")
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)
		return model.NewValidationError(field, "unknown field")
	case strings.Contains(err.Error(), "UUID"):
		return model.NewValidationError("body", "identifiers must be valid UUIDs")
	default:
		return model.NewValidationError("body", "invalid JSON: "+err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
