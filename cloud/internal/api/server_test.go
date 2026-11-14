package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const token = "test-token-0123456789"

func do(t *testing.T, h http.Handler, method, path, body, auth string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAPI(t *testing.T) {
	c, _ := newCloud(10)
	h := NewServer(c, token).Handler()

	if rec := do(t, h, "GET", "/v1/networks", "", ""); rec.Code != 401 {
		t.Fatalf("no token: want 401, got %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/v1/networks", "", "wrong"); rec.Code != 401 {
		t.Fatalf("bad token: want 401, got %d", rec.Code)
	}

	rec := do(t, h, "POST", "/v1/networks", `{"name":"net"}`, token)
	if rec.Code != 201 {
		t.Fatalf("create network: %d %s", rec.Code, rec.Body)
	}
	var n Network
	json.Unmarshal(rec.Body.Bytes(), &n)

	if rec := do(t, h, "POST", "/v1/networks", `{"name":"x","bogus":1}`, token); rec.Code != 400 {
		t.Fatalf("unknown field: want 400, got %d", rec.Code)
	}

	rec = do(t, h, "POST", "/v1/networks", `{"name":"net"}`, token)
	var conflict struct {
		Error    string  `json:"error"`
		Existing Network `json:"existing"`
	}
	json.Unmarshal(rec.Body.Bytes(), &conflict)
	if rec.Code != 409 || conflict.Existing.ID != n.ID {
		t.Fatalf("duplicate: want 409 with existing, got %d %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "POST", "/v1/instances", `{"name":"vm","networkId":"`+n.ID+`","size":"small"}`, token)
	if rec.Code != 201 {
		t.Fatalf("create instance: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/v1/instances/vm-missing", "", token); rec.Code != 404 {
		t.Fatalf("missing: want 404, got %d", rec.Code)
	}
	if rec := do(t, h, "DELETE", "/v1/networks/"+n.ID, "", token); rec.Code != 409 {
		t.Fatalf("delete in-use network: want 409, got %d", rec.Code)
	}
}

func TestFaultInjection(t *testing.T) {
	c, _ := newCloud(10)
	h := NewServer(c, token).Handler()
	if rec := do(t, h, "PUT", "/admin/faults", `{"errorRate":1}`, token); rec.Code != 200 {
		t.Fatalf("set faults: %d", rec.Code)
	}
	if rec := do(t, h, "POST", "/v1/networks", `{"name":"n"}`, token); rec.Code != 503 {
		t.Fatalf("want injected 503, got %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/v1/networks", "", token); rec.Code != 200 {
		t.Fatalf("reads are never faulted, got %d", rec.Code)
	}
	if rec := do(t, h, "PUT", "/admin/faults", `{"errorRate":2}`, token); rec.Code != 400 {
		t.Fatalf("invalid fault config: want 400, got %d", rec.Code)
	}
	rec := do(t, h, "GET", "/metrics", "", "")
	if !strings.Contains(rec.Body.String(), `gantry_cloud_requests_total{method="POST",route="/v1/networks",code="503"} 1`) {
		t.Fatalf("metrics missing faulted request:\n%s", rec.Body)
	}
}
