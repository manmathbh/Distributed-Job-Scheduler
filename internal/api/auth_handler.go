package api

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/manmathbh/distributed-job-scheduler/internal/auth"
)

// RegisterRequest is the public client registration payload.
// API-key authentication is the credential model for this scheduler, so
// registration creates a client API key instead of a password account.
type RegisterRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// RegisterResponse contains the one-time credential returned after registration.
type RegisterResponse struct {
	UserID  string   `json:"user_id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	APIKey  string   `json:"api_key"`
	Type    string   `json:"type"`
	Scopes  []string `json:"scopes"`
	Message string   `json:"message"`
}

// handleRegister creates a new client principal and returns its API key.
// This endpoint is intentionally public; all scheduler endpoints remain
// protected by the existing Bearer API-key middleware.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErr(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req RegisterRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if len(req.Name) < 2 || len(req.Name) > 80 {
		s.writeErr(w, http.StatusBadRequest, "invalid_name", "name must be between 2 and 80 characters")
		return
	}

	if req.Email == "" || len(req.Email) > 254 {
		s.writeErr(w, http.StatusBadRequest, "invalid_email", "a valid email is required")
		return
	}
	parsed, err := mail.ParseAddress(req.Email)
	if err != nil || parsed.Address != req.Email {
		s.writeErr(w, http.StatusBadRequest, "invalid_email", "a valid email is required")
		return
	}

	// API keys are the identity/credential for this service. The email is
	// accepted for the registration UX and returned to the client, while the
	// stable owner ID is an opaque UUID and is what project authorization uses.
	ownerID := "user_" + uuid.NewString()
	key, err := auth.GenerateAPIKey(req.Name, ownerID, auth.KeyTypeClient)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "key_generation_failed", "failed to generate client API key")
		return
	}

	if err := s.authStore.CreateKey(r.Context(), key); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "key_storage_failed", "failed to create client account")
		return
	}

	s.writeJSON(w, http.StatusCreated, RegisterResponse{
		UserID:  ownerID,
		Name:    req.Name,
		Email:   req.Email,
		APIKey:  key.Key,
		Type:    string(key.Type),
		Scopes:  key.Scopes,
		Message: "Registration successful. Save your API key; it is your scheduler credential.",
	})
}
