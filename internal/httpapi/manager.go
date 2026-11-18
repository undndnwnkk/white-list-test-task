package httpapi

import (
	"net/http"
)

func (h *Handler) getManagerSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	schedule, err := h.svc.GetManagerSchedule(r.Context(), id, r.URL.Query().Get("date"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, schedule)
}
