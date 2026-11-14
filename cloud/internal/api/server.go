package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Faults are injected into mutating /v1 calls so clients can be tested
// against an unreliable API on purpose.
type Faults struct {
	ErrorRate float64 `json:"errorRate"` // 0..1 chance of a 503
	LatencyMs int     `json:"latencyMs"`
}

type Server struct {
	cloud *Cloud
	token string

	mu       sync.Mutex
	faults   Faults
	requests map[string]int // "method route code" -> count
}

func NewServer(c *Cloud, token string) *Server {
	return &Server{cloud: c, token: token, requests: map[string]int{}}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("GET /metrics", s.metrics)

	api := http.NewServeMux()
	api.HandleFunc("POST /v1/networks", s.createNetwork)
	api.HandleFunc("GET /v1/networks", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, s.cloud.ListNetworks()) })
	api.HandleFunc("GET /v1/networks/{id}", s.getNetwork)
	api.HandleFunc("DELETE /v1/networks/{id}", s.deleteNetwork)
	api.HandleFunc("POST /v1/instances", s.createInstance)
	api.HandleFunc("GET /v1/instances", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, s.cloud.ListInstances()) })
	api.HandleFunc("GET /v1/instances/{id}", s.getInstance)
	api.HandleFunc("PATCH /v1/instances/{id}", s.resizeInstance)
	api.HandleFunc("DELETE /v1/instances/{id}", s.deleteInstance)
	api.HandleFunc("PUT /admin/faults", s.setFaults)
	api.HandleFunc("GET /admin/faults", func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, 200, s.faults)
	})

	mux.Handle("/v1/", s.auth(s.inject(api, api)))
	mux.Handle("/admin/", s.auth(api))
	return s.count(mux)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid API token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) inject(routes *http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		f := s.faults
		s.mu.Unlock()
		if f.LatencyMs > 0 {
			time.Sleep(time.Duration(f.LatencyMs) * time.Millisecond)
		}
		if r.Method != http.MethodGet && f.ErrorRate > 0 && rand.Float64() < f.ErrorRate {
			_, r.Pattern = routes.Handler(r) // label the metric with the real route
			writeError(w, http.StatusServiceUnavailable, "unavailable", "injected fault: try again")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type recorder struct {
	http.ResponseWriter
	code int
}

func (r *recorder) WriteHeader(code int) { r.code = code; r.ResponseWriter.WriteHeader(code) }

func (s *Server) count(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recorder{ResponseWriter: w, code: 200}
		next.ServeHTTP(rec, r)
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		if _, after, ok := strings.Cut(route, " "); ok {
			route = after
		}
		s.mu.Lock()
		s.requests[fmt.Sprintf("%s %s %d", r.Method, route, rec.code)]++
		s.mu.Unlock()
	})
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	nets, states := s.cloud.Counts()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintln(w, "# HELP gantry_cloud_networks Networks that exist.")
	fmt.Fprintln(w, "# TYPE gantry_cloud_networks gauge")
	fmt.Fprintf(w, "gantry_cloud_networks %d\n", nets)
	fmt.Fprintln(w, "# HELP gantry_cloud_instances Instances by state.")
	fmt.Fprintln(w, "# TYPE gantry_cloud_instances gauge")
	for _, st := range []string{"provisioning", "running", "resizing"} {
		fmt.Fprintf(w, "gantry_cloud_instances{state=%q} %d\n", st, states[st])
	}
	fmt.Fprintln(w, "# HELP gantry_cloud_requests_total API requests by method, route and status code.")
	fmt.Fprintln(w, "# TYPE gantry_cloud_requests_total counter")
	s.mu.Lock()
	keys := make([]string, 0, len(s.requests))
	for k := range s.requests {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var method, route string
		var code int
		fmt.Sscanf(k, "%s %s %d", &method, &route, &code)
		fmt.Fprintf(w, "gantry_cloud_requests_total{method=%q,route=%q,code=\"%d\"} %d\n", method, route, code, s.requests[k])
	}
	s.mu.Unlock()
}

// ---- handlers ----

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid", "request body: "+err.Error())
		return false
	}
	return true
}

func (s *Server) createNetwork(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
		CIDR string `json:"cidr"`
	}
	if !decode(w, r, &in) {
		return
	}
	n, err := s.cloud.CreateNetwork(in.Name, in.CIDR)
	respond(w, http.StatusCreated, n, err)
}

func (s *Server) getNetwork(w http.ResponseWriter, r *http.Request) {
	n, err := s.cloud.GetNetwork(r.PathValue("id"))
	respond(w, http.StatusOK, n, err)
}

func (s *Server) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusNoContent, nil, s.cloud.DeleteNetwork(r.PathValue("id")))
}

func (s *Server) createInstance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string `json:"name"`
		NetworkID string `json:"networkId"`
		Size      string `json:"size"`
	}
	if !decode(w, r, &in) {
		return
	}
	i, err := s.cloud.CreateInstance(in.Name, in.NetworkID, in.Size)
	respond(w, http.StatusCreated, i, err)
}

func (s *Server) getInstance(w http.ResponseWriter, r *http.Request) {
	i, err := s.cloud.GetInstance(r.PathValue("id"))
	respond(w, http.StatusOK, i, err)
}

func (s *Server) resizeInstance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Size string `json:"size"`
	}
	if !decode(w, r, &in) {
		return
	}
	i, err := s.cloud.ResizeInstance(r.PathValue("id"), in.Size)
	respond(w, http.StatusOK, i, err)
}

func (s *Server) deleteInstance(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusNoContent, nil, s.cloud.DeleteInstance(r.PathValue("id")))
}

func (s *Server) setFaults(w http.ResponseWriter, r *http.Request) {
	var f Faults
	if !decode(w, r, &f) {
		return
	}
	if f.ErrorRate < 0 || f.ErrorRate > 1 || f.LatencyMs < 0 || f.LatencyMs > 10_000 {
		writeError(w, http.StatusBadRequest, "invalid", "errorRate must be 0..1, latencyMs 0..10000")
		return
	}
	s.mu.Lock()
	s.faults = f
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, f)
}

var statusFor = map[error]int{
	ErrNotFound:  http.StatusNotFound,
	ErrConflict:  http.StatusConflict,
	ErrInUse:     http.StatusConflict,
	ErrOverlap:   http.StatusConflict,
	ErrExhausted: http.StatusConflict,
	ErrBusy:      http.StatusConflict,
	ErrQuota:     http.StatusForbidden,
	ErrInvalid:   http.StatusBadRequest,
}

// respond writes v on success. A conflict on create still returns the
// existing object, so clients can adopt it idempotently.
func respond(w http.ResponseWriter, ok int, v any, err error) {
	if err == nil {
		if v == nil {
			w.WriteHeader(ok)
			return
		}
		writeJSON(w, ok, v)
		return
	}
	for e, code := range statusFor {
		if errors.Is(err, e) {
			body := map[string]any{"error": e.Error(), "message": message(e)}
			if errors.Is(err, ErrConflict) && v != nil {
				body["existing"] = v
			}
			writeJSON(w, code, body)
			return
		}
	}
	writeError(w, http.StatusInternalServerError, "internal", err.Error())
}

func message(e error) string {
	switch e {
	case ErrNotFound:
		return "resource does not exist"
	case ErrConflict:
		return "a resource with this name already exists"
	case ErrInUse:
		return "network still has instances"
	case ErrOverlap:
		return "CIDR overlaps an existing network"
	case ErrExhausted:
		return "no free addresses left"
	case ErrBusy:
		return "instance must be running to resize"
	case ErrQuota:
		return "instance quota exceeded"
	default:
		return "invalid request"
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, kind, msg string) {
	writeJSON(w, code, map[string]string{"error": kind, "message": msg})
}
