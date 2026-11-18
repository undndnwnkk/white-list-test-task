package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
	"github.com/undndnwnkk/white-list-test-task/internal/service"
)

var (
	testNow    = time.Date(2030, 1, 6, 12, 0, 0, 0, time.UTC)
	discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))
)

func decodeErrorBody(t *testing.T, body io.Reader) errorBody {
	t.Helper()
	var resp errorResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return resp.Error
}

func TestRequestsRejectedBeforeDatabase(t *testing.T) {
	router := NewRouter(service.New(nil, func() time.Time { return testNow }), discardLog)
	id := uuid.NewString()
	meetingBody := func(start string) string {
		return fmt.Sprintf(`{"company_id":%q,"manager_id":%q,"start_at":%q}`, id, id, start)
	}

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
		wantField  string
	}{
		{"malformed json", "POST", "/companies", `{"name":`, 400, "validation_error", "body"},
		{"empty body", "POST", "/companies", ``, 400, "validation_error", "body"},
		{"two json objects", "POST", "/companies", `{} {}`, 400, "validation_error", "body"},
		{"unknown field", "POST", "/companies", `{"name":"A","bin":"170340021554","phone":"+77011234567","extra":1}`, 400, "validation_error", "extra"},
		{"wrong field type", "POST", "/companies", `{"name":1}`, 400, "validation_error", "name"},
		{"invalid bin", "POST", "/companies", `{"name":"A","bin":"2l0640900180","phone":"+77011234567"}`, 400, "validation_error", "bin"},
		{"empty name", "POST", "/companies", `{"name":"  ","bin":"170340021554","phone":"+77011234567"}`, 400, "validation_error", "name"},
		{"invalid uuid in body", "POST", "/meetings", `{"company_id":"abc","manager_id":"abc","start_at":"2030-01-07T05:00:00Z"}`, 400, "validation_error", "body"},
		{"time not in ISO 8601", "POST", "/meetings", meetingBody("2030-01-07 10:00"), 400, "validation_error", "body"},
		{"meeting at 17:30 Almaty", "POST", "/meetings", meetingBody("2030-01-07T12:30:00Z"), 422, "outside_working_hours", ""},
		{"meeting at 08:59 Almaty", "POST", "/meetings", meetingBody("2030-01-07T03:59:00Z"), 422, "outside_working_hours", ""},
		{"meeting on saturday", "POST", "/meetings", meetingBody("2030-01-12T05:00:00Z"), 422, "outside_working_hours", ""},
		{"meeting in the past", "POST", "/meetings", meetingBody("2029-01-08T05:00:00Z"), 422, "meeting_in_past", ""},
		{"meeting without start", "POST", "/meetings", fmt.Sprintf(`{"company_id":%q,"manager_id":%q}`, id, id), 400, "validation_error", "start_at"},
		{"patch with invalid id", "PATCH", "/meetings/not-a-uuid", `{"status":"cancelled"}`, 400, "validation_error", "id"},
		{"patch with empty body", "PATCH", "/meetings/" + id, `{}`, 400, "validation_error", "body"},
		{"patch to unsupported status", "PATCH", "/meetings/" + id, `{"status":"scheduled"}`, 400, "validation_error", "status"},
		{"get meeting with invalid id", "GET", "/meetings/123", ``, 400, "validation_error", "id"},
		{"limit not a number", "GET", "/companies?limit=abc", ``, 400, "validation_error", "limit"},
		{"limit too big", "GET", "/companies?limit=1000", ``, 400, "validation_error", "limit"},
		{"negative offset", "GET", "/companies?offset=-1", ``, 400, "validation_error", "offset"},
		{"unknown status filter", "GET", "/companies?status=active", ``, 400, "validation_error", "status"},
		{"schedule with bad date", "GET", "/managers/" + id + "/schedule?date=07.01.2030", ``, 400, "validation_error", "date"},
		{"schedule without date", "GET", "/managers/" + id + "/schedule", ``, 400, "validation_error", "date"},
		{"meeting_set call without meeting", "POST", "/calls", fmt.Sprintf(`{"company_id":%q,"manager_id":%q,"result":"meeting_set"}`, id, id), 400, "validation_error", "meeting_id"},
		{"unknown call result", "POST", "/calls", fmt.Sprintf(`{"company_id":%q,"manager_id":%q,"result":"busy"}`, id, id), 400, "validation_error", "result"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("content type = %q, want JSON", ct)
			}
			body := decodeErrorBody(t, rec.Body)
			if body.Code != tc.wantCode || body.Field != tc.wantField || body.Message == "" {
				t.Fatalf("error = %+v, want code %q field %q", body, tc.wantCode, tc.wantField)
			}
		})
	}
}

func TestResolveErrorMapping(t *testing.T) {
	for _, m := range errorMappings {
		t.Run(m.code, func(t *testing.T) {
			status, body := resolveError(fmt.Errorf("wrapped: %w", m.err))
			if status != m.status || body.Code != m.code || body.Message != m.err.Error() {
				t.Fatalf("got %d %+v, want %d %q", status, body, m.status, m.code)
			}
		})
	}

	t.Run("database details are hidden", func(t *testing.T) {
		status, body := resolveError(errors.Join(model.ErrInvalidData, errors.New("pq: secret details")))
		if status != http.StatusUnprocessableEntity || strings.Contains(body.Message, "secret") {
			t.Fatalf("got %d %+v", status, body)
		}
	})

	t.Run("unknown error is 500 without details", func(t *testing.T) {
		status, body := resolveError(errors.New("connection refused"))
		if status != http.StatusInternalServerError || body.Code != "internal_error" || strings.Contains(body.Message, "refused") {
			t.Fatalf("got %d %+v", status, body)
		}
	})
}

func TestRecoverPanicsReturnsJSON(t *testing.T) {
	h := &Handler{log: discardLog}
	handler := h.recoverPanics(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if body := decodeErrorBody(t, rec.Body); body.Code != "internal_error" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
