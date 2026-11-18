package httpapi

import (
	"net/http"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func (h *Handler) createCompany(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCompanyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.writeError(w, r, err)
		return
	}

	company, err := h.svc.CreateCompany(r.Context(), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, company)
}

func (h *Handler) listCompanies(w http.ResponseWriter, r *http.Request) {
	var filter model.ListCompaniesFilter

	if raw := r.URL.Query().Get("status"); raw != "" {
		status := model.CompanyStatus(raw)
		filter.Status = &status
	}

	var err error
	if filter.Limit, err = queryInt(r, "limit"); err != nil {
		h.writeError(w, r, err)
		return
	}
	if filter.Offset, err = queryInt(r, "offset"); err != nil {
		h.writeError(w, r, err)
		return
	}

	list, err := h.svc.ListCompanies(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
