package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/manmathbh/distributed-job-scheduler/internal/auth"
	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/queue"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/manmathbh/distributed-job-scheduler/internal/wal"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type v1TestServer struct {
	Handler http.Handler
	Admin   string
	OwnerA  string
	OwnerB  string
	Svc     *service.Service
}

func setupV1Server(t *testing.T) *v1TestServer {
	t.Helper()
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	authStore := auth.NewRedisStore(redisClient)

	ctx := context.Background()
	mk := func(name, owner string, kt auth.KeyType) string {
		k, err := auth.GenerateAPIKey(name, owner, kt)
		require.NoError(t, err)
		require.NoError(t, authStore.CreateKey(ctx, k))
		return k.Key
	}
	adminKey := mk("admin", "admin", auth.KeyTypeAdmin)
	ownerA := mk("client-a", "owner-a", auth.KeyTypeClient)
	ownerB := mk("client-b", "owner-b", auth.KeyTypeClient)

	w, err := wal.Open(t.TempDir())
	require.NoError(t, err)
	core, err := queue.NewCore(w)
	require.NoError(t, err)

	st := store.NewMemoryStore()
	svc := service.New(st, metrics.NewRegistry())

	server := NewServer(core, 30*time.Second, authStore, WithService(svc), WithMetrics(metrics.NewRegistry()))
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	return &v1TestServer{Handler: mux, Admin: adminKey, OwnerA: ownerA, OwnerB: ownerB, Svc: svc}
}

func (s *v1TestServer) do(t *testing.T, method, path, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.Handler.ServeHTTP(rr, req)
	return rr
}

func decodeMap(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &m))
	return m
}

func TestV1_Projects_CRUD_And_Authz(t *testing.T) {
	s := setupV1Server(t)

	// Unauthenticated.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	s.Handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)

	// Create project as owner-a.
	rr = s.do(t, "POST", "/api/v1/projects", s.OwnerA, map[string]any{"name": "Alpha"})
	require.Equal(t, http.StatusCreated, rr.Code)
	proj := decodeMap(t, rr)
	id := proj["id"].(string)

	// owner-b cannot read owner-a's project.
	rr = s.do(t, "GET", "/api/v1/projects/"+id, s.OwnerB, nil)
	require.Equal(t, http.StatusForbidden, rr.Code)

	// admin can read any project.
	rr = s.do(t, "GET", "/api/v1/projects/"+id, s.Admin, nil)
	require.Equal(t, http.StatusOK, rr.Code)

	// owner-a can read it.
	rr = s.do(t, "GET", "/api/v1/projects/"+id, s.OwnerA, nil)
	require.Equal(t, http.StatusOK, rr.Code)

	// List scoped to owner-a.
	rr = s.do(t, "GET", "/api/v1/projects", s.OwnerA, nil)
	require.Equal(t, http.StatusOK, rr.Code)
	items := decodeMap(t, rr)["items"].([]any)
	require.Len(t, items, 1)

	// owner-b sees none.
	rr = s.do(t, "GET", "/api/v1/projects", s.OwnerB, nil)
	items = decodeMap(t, rr)["items"].([]any)
	require.Len(t, items, 0)
}

func TestV1_Queues_Lifecycle(t *testing.T) {
	s := setupV1Server(t)
	rr := s.do(t, "POST", "/api/v1/projects", s.OwnerA, map[string]any{"name": "P"})
	projID := decodeMap(t, rr)["id"].(string)

	// Create queue with retry config.
	rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues", s.OwnerA, map[string]any{
		"name": "default", "concurrency": 3, "retry_strategy": "exponential", "max_attempts": 5,
	})
	require.Equal(t, http.StatusCreated, rr.Code)
	q := decodeMap(t, rr)
	queueID := q["id"].(string)
	require.Equal(t, float64(3), q["concurrency"])

	// Pause.
	rr = s.do(t, "POST", "/api/v1/queues/"+queueID+"/pause", s.OwnerA, nil)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "paused", decodeMap(t, rr)["status"])

	// Resume.
	rr = s.do(t, "POST", "/api/v1/queues/"+queueID+"/resume", s.OwnerA, nil)
	require.Equal(t, "active", decodeMap(t, rr)["status"])

	// Stats.
	rr = s.do(t, "GET", "/api/v1/queues/"+queueID+"/stats", s.OwnerA, nil)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, float64(0), decodeMap(t, rr)["total"])
}

func TestV1_Jobs_SubmitListGet(t *testing.T) {
	s := setupV1Server(t)
	rr := s.do(t, "POST", "/api/v1/projects", s.OwnerA, map[string]any{"name": "P"})
	projID := decodeMap(t, rr)["id"].(string)
	rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues", s.OwnerA, map[string]any{"name": "q"})
	queueID := decodeMap(t, rr)["id"].(string)

	// Submit immediate job.
	rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues/"+queueID+"/jobs", s.OwnerA, map[string]any{
		"type": "immediate", "payload": map[string]any{"url": "https://example.com"},
	})
	require.Equal(t, http.StatusCreated, rr.Code)
	job := decodeMap(t, rr)
	jobID := job["id"].(string)
	require.Equal(t, "queued", job["status"])

	// Get job.
	rr = s.do(t, "GET", "/api/v1/jobs/"+jobID, s.OwnerA, nil)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, jobID, decodeMap(t, rr)["id"])

	// Scheduled job without scheduled_at must be rejected.
	rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues/"+queueID+"/jobs", s.OwnerA, map[string]any{"type": "scheduled"})
	require.Equal(t, http.StatusBadRequest, rr.Code)

	// List jobs.
	rr = s.do(t, "GET", "/api/v1/projects/"+projID+"/jobs?limit=10", s.OwnerA, nil)
	require.Equal(t, http.StatusOK, rr.Code)
	items := decodeMap(t, rr)["items"].([]any)
	require.Len(t, items, 1)
}

func TestV1_Jobs_Pagination(t *testing.T) {
	s := setupV1Server(t)
	rr := s.do(t, "POST", "/api/v1/projects", s.OwnerA, map[string]any{"name": "P"})
	projID := decodeMap(t, rr)["id"].(string)
	rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues", s.OwnerA, map[string]any{"name": "q"})
	queueID := decodeMap(t, rr)["id"].(string)

	for i := 0; i < 15; i++ {
		rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues/"+queueID+"/jobs", s.OwnerA, map[string]any{"type": "immediate"})
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	seen := map[string]bool{}
	cursor := ""
	pages := 0
	for {
		url := "/api/v1/projects/" + projID + "/jobs?limit=7"
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		rr = s.do(t, "GET", url, s.OwnerA, nil)
		require.Equal(t, http.StatusOK, rr.Code)
		body := decodeMap(t, rr)
		for _, it := range body["items"].([]any) {
			id := it.(map[string]any)["id"].(string)
			require.False(t, seen[id], "duplicate job %s across pages", id)
			seen[id] = true
		}
		pages++
		cursor, _ = body["next_cursor"].(string)
		if cursor == "" {
			break
		}
	}
	require.Equal(t, 15, len(seen))
	require.GreaterOrEqual(t, pages, 3)
}

func TestV1_RecurringSchedule(t *testing.T) {
	s := setupV1Server(t)
	rr := s.do(t, "POST", "/api/v1/projects", s.OwnerA, map[string]any{"name": "P"})
	projID := decodeMap(t, rr)["id"].(string)
	rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues", s.OwnerA, map[string]any{"name": "q"})
	queueID := decodeMap(t, rr)["id"].(string)

	rr = s.do(t, "POST", "/api/v1/projects/"+projID+"/queues/"+queueID+"/jobs", s.OwnerA, map[string]any{
		"type": "recurring", "cron_expr": "*/5 * * * *", "payload": map[string]any{"k": "v"},
	})
	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "recurring", decodeMap(t, rr)["type"])
}
