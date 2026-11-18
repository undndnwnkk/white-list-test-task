package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/undndnwnkk/white-list-test-task/internal/service"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewRouter(svc *service.Service, log *slog.Logger) http.Handler {
	h := &Handler{svc: svc, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /companies", h.createCompany)
	mux.HandleFunc("GET /companies", h.listCompanies)
	mux.HandleFunc("POST /calls", h.createCall)
	mux.HandleFunc("POST /meetings", h.createMeeting)
	mux.HandleFunc("GET /meetings/{id}", h.getMeeting)
	mux.HandleFunc("PATCH /meetings/{id}", h.updateMeeting)
	mux.HandleFunc("GET /managers/{id}/schedule", h.getManagerSchedule)

	return h.logRequests(h.recoverPanics(mux))
}
