package httpapi

import (
	"net/http"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func (h *Handler) createMeeting(w http.ResponseWriter, r *http.Request) {
	var req model.CreateMeetingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)
		return
	}

	meeting, err := h.svc.CreateMeeting(r.Context(), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, meeting)
}

func (h *Handler) getMeeting(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	meeting, err := h.svc.GetMeeting(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, meeting)
}

func (h *Handler) updateMeeting(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	var req model.UpdateMeetingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)
		return
	}

	meeting, err := h.svc.UpdateMeeting(r.Context(), id, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, meeting)
}
