package httpapi

import (
	"net/http"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func (h *Handler) createCall(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCallRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)
		return
	}

	call, err := h.svc.CreateCall(r.Context(), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, call)
}
