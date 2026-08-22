package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/manmathbh/distributed-job-scheduler/internal/auth"
	"github.com/stretchr/testify/require"
)

func newTestAuthStore(t *testing.T) auth.Store {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return auth.NewRedisStore(client)
}

func TestHandleRegisterCreatesClientAPIKey(t *testing.T) {
	store := newTestAuthStore(t)
	s := NewServer(nil, 0, store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", stringsReader(`{"name":"Test User","email":"test@example.com"}`))
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	s.handleRegister(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var got RegisterResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	require.Equal(t, "Test User", got.Name)
	require.Equal(t, "test@example.com", got.Email)
	require.Equal(t, "client", got.Type)
	require.NotEmpty(t, got.UserID)
	require.NotEmpty(t, got.APIKey)
	require.Contains(t, got.APIKey, "client_")

	stored, err := store.GetKey(context.Background(), got.APIKey)
	require.NoError(t, err)
	require.Equal(t, got.UserID, stored.OwnerID)
	require.Equal(t, auth.KeyTypeClient, stored.Type)
}

func TestHandleRegisterRejectsInvalidEmail(t *testing.T) {
	store := newTestAuthStore(t)
	s := NewServer(nil, 0, store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", stringsReader(`{"name":"Test User","email":"not-an-email"}`))
	rec := httptest.NewRecorder()

	s.handleRegister(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// stringsReader keeps the tests independent of io/ioutil helpers.
func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }
