package httpapi

import (
	"errors"
	"net/http"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorMapping struct {
	err    error
	status int
	code   string
}

var errorMappings = []errorMapping{
	{model.ErrCompanyNotFound, http.StatusNotFound, "company_not_found"},
	{model.ErrManagerNotFound, http.StatusNotFound, "manager_not_found"},
	{model.ErrMeetingNotFound, http.StatusNotFound, "meeting_not_found"},

	{model.ErrBINAlreadyExists, http.StatusConflict, "bin_already_exists"},
	{model.ErrManagerBusy, http.StatusConflict, "manager_busy"},
	{model.ErrCompanyBusy, http.StatusConflict, "company_busy"},
	{model.ErrCompanyHasActiveMeeting, http.StatusConflict, "company_has_active_meeting"},
	{model.ErrMeetingCancelled, http.StatusConflict, "meeting_cancelled"},
	{model.ErrMeetingAlreadyStarted, http.StatusConflict, "meeting_already_started"},

	{model.ErrOutsideWorkingHours, http.StatusUnprocessableEntity, "outside_working_hours"},
	{model.ErrMeetingInPast, http.StatusUnprocessableEntity, "meeting_in_past"},
	{model.ErrMeetingCompanyMismatch, http.StatusUnprocessableEntity, "meeting_company_mismatch"},
	{model.ErrInvalidData, http.StatusUnprocessableEntity, "invalid_data"},
}

func resolveError(err error) (int, errorBody) {
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest, errorBody{
			Code:    "validation_error",
			Message: validationErr.Message,
			Field:   validationErr.Field,
		}
	}

	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			return m.status, errorBody{Code: m.code, Message: m.err.Error()}
		}
	}

	return http.StatusInternalServerError, errorBody{Code: "internal_error", Message: "internal server error"}
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, body := resolveError(err)
	if status == http.StatusInternalServerError {
		h.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	}
	writeJSON(w, status, errorResponse{Error: body})
}
