package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/undndnwnkk/white-list-test-task/internal/repository"
	"github.com/undndnwnkk/white-list-test-task/internal/service"
	"github.com/undndnwnkk/white-list-test-task/internal/testdb"
)

const (
	managerA = "11111111-1111-1111-1111-111111111111"
	managerB = "22222222-2222-2222-2222-222222222222"
)

type apiClient struct {
	t      *testing.T
	server *httptest.Server
}

type apiResponse struct {
	status int
	body   map[string]any
}

func setupAPI(t *testing.T) *apiClient {
	t.Helper()
	pool := testdb.Open(t, "httpapi")

	svc := service.New(repository.New(pool), func() time.Time { return testNow })
	server := httptest.NewServer(NewRouter(svc, discardLog))
	t.Cleanup(server.Close)

	return &apiClient{t: t, server: server}
}

func (c *apiClient) do(method, path string, body any) apiResponse {
	c.t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.server.URL+path, reader)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	resp, err := c.server.Client().Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	result := apiResponse{status: resp.StatusCode}
	if err := json.NewDecoder(resp.Body).Decode(&result.body); err != nil {
		c.t.Fatalf("%s %s: decode body: %v", method, path, err)
	}
	return result
}

func (c *apiClient) expect(resp apiResponse, status int, code string) apiResponse {
	c.t.Helper()
	if resp.status != status {
		c.t.Fatalf("status = %d, want %d, body: %v", resp.status, status, resp.body)
	}
	if code != "" {
		errBody, _ := resp.body["error"].(map[string]any)
		if errBody["code"] != code {
			c.t.Fatalf("error code = %v, want %q, body: %v", errBody["code"], code, resp.body)
		}
	}
	return resp
}

func (c *apiClient) createCompany(name, bin string) string {
	c.t.Helper()
	resp := c.expect(c.do("POST", "/companies", map[string]string{
		"name": name, "bin": bin, "phone": "+7 701 123 45 67",
	}), http.StatusCreated, "")
	return resp.body["id"].(string)
}

func TestMeetingLifecycleOverHTTP(t *testing.T) {
	api := setupAPI(t)

	company1 := api.createCompany("Алматы Трейд", "170340021554")
	company2 := api.createCompany("Digital Solutions", "170340021555")
	api.expect(api.do("POST", "/companies", map[string]string{
		"name": "Duplicate", "bin": "170340021554", "phone": "+77011234567",
	}), http.StatusConflict, "bin_already_exists")

	m1 := api.expect(api.do("POST", "/meetings", map[string]string{
		"company_id": company1, "manager_id": managerA, "start_at": "2030-01-07T05:00:00Z",
	}), http.StatusCreated, "")
	if m1.body["start_at"] != "2030-01-07T05:00:00Z" || m1.body["end_at"] != "2030-01-07T06:00:00Z" {
		t.Fatalf("times must be returned in UTC: %v", m1.body)
	}
	meeting1 := m1.body["id"].(string)

	api.expect(api.do("POST", "/meetings", map[string]string{
		"company_id": company2, "manager_id": managerA, "start_at": "2030-01-07T05:30:00Z",
	}), http.StatusConflict, "manager_busy")

	m2 := api.expect(api.do("POST", "/meetings", map[string]string{
		"company_id": company2, "manager_id": managerA, "start_at": "2030-01-07T06:00:00Z",
	}), http.StatusCreated, "")
	meeting2 := m2.body["id"].(string)

	api.expect(api.do("POST", "/meetings", map[string]string{
		"company_id": company1, "manager_id": managerB, "start_at": "2030-01-08T05:00:00Z",
	}), http.StatusConflict, "company_has_active_meeting")

	api.expect(api.do("POST", "/calls", map[string]string{
		"company_id": company1, "manager_id": managerA, "result": "meeting_set",
	}), http.StatusBadRequest, "validation_error")
	api.expect(api.do("POST", "/calls", map[string]string{
		"company_id": company1, "manager_id": managerA, "result": "meeting_set", "meeting_id": meeting2,
	}), http.StatusUnprocessableEntity, "meeting_company_mismatch")
	api.expect(api.do("POST", "/calls", map[string]string{
		"company_id": company1, "manager_id": managerA, "result": "meeting_set", "meeting_id": meeting1,
	}), http.StatusCreated, "")
	api.expect(api.do("POST", "/calls", map[string]string{
		"company_id": company2, "manager_id": managerA, "result": "no_answer",
	}), http.StatusCreated, "")

	list := api.expect(api.do("GET", "/companies?status=meeting_set", nil), http.StatusOK, "")
	if list.body["total"] != float64(1) {
		t.Fatalf("expected one company with meeting_set: %v", list.body)
	}

	moved := api.expect(api.do("PATCH", "/meetings/"+meeting1, map[string]string{
		"start_at": "2030-01-08T09:00:00Z",
	}), http.StatusOK, "")
	if moved.body["original_start_at"] != "2030-01-07T05:00:00Z" || moved.body["start_at"] != "2030-01-08T09:00:00Z" {
		t.Fatalf("reschedule must keep original start: %v", moved.body)
	}
	if history, _ := moved.body["history"].([]any); len(history) != 2 {
		t.Fatalf("expected 2 history events: %v", moved.body["history"])
	}

	api.expect(api.do("PATCH", "/meetings/"+meeting1, map[string]string{
		"start_at": "2030-01-08T12:30:00Z",
	}), http.StatusUnprocessableEntity, "outside_working_hours")

	monday := api.expect(api.do("GET", "/managers/"+managerA+"/schedule?date=2030-01-07", nil), http.StatusOK, "")
	if meetings, _ := monday.body["meetings"].([]any); len(meetings) != 1 {
		t.Fatalf("expected only meeting2 on monday: %v", monday.body)
	}
	tuesday := api.expect(api.do("GET", "/managers/"+managerA+"/schedule?date=2030-01-08", nil), http.StatusOK, "")
	if meetings, _ := tuesday.body["meetings"].([]any); len(meetings) != 1 {
		t.Fatalf("expected moved meeting1 on tuesday: %v", tuesday.body)
	}

	api.expect(api.do("PATCH", "/meetings/"+meeting1, map[string]string{"status": "cancelled"}), http.StatusOK, "")
	api.expect(api.do("PATCH", "/meetings/"+meeting1, map[string]string{"status": "cancelled"}), http.StatusConflict, "meeting_cancelled")

	full := api.expect(api.do("GET", "/meetings/"+meeting1, nil), http.StatusOK, "")
	if history, _ := full.body["history"].([]any); len(history) != 3 {
		t.Fatalf("expected created, rescheduled, cancelled events: %v", full.body["history"])
	}
}

func TestNotFoundResponses(t *testing.T) {
	api := setupAPI(t)
	unknown := "99999999-9999-9999-9999-999999999999"

	api.expect(api.do("GET", "/meetings/"+unknown, nil), http.StatusNotFound, "meeting_not_found")
	api.expect(api.do("PATCH", "/meetings/"+unknown, map[string]string{"status": "cancelled"}), http.StatusNotFound, "meeting_not_found")
	api.expect(api.do("GET", "/managers/"+unknown+"/schedule?date=2030-01-07", nil), http.StatusNotFound, "manager_not_found")
	api.expect(api.do("POST", "/meetings", map[string]string{
		"company_id": unknown, "manager_id": managerA, "start_at": "2030-01-07T05:00:00Z",
	}), http.StatusNotFound, "company_not_found")

	company := api.createCompany("Test", "100000000001")
	api.expect(api.do("POST", "/meetings", map[string]string{
		"company_id": company, "manager_id": unknown, "start_at": "2030-01-07T05:00:00Z",
	}), http.StatusNotFound, "manager_not_found")
	api.expect(api.do("POST", "/calls", map[string]string{
		"company_id": company, "manager_id": managerA, "result": "meeting_set", "meeting_id": unknown,
	}), http.StatusNotFound, "meeting_not_found")
}
