package auth

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/ford-retain/backend/internal/platform/httpx"
	"github.com/ford-retain/backend/internal/platform/identity"
)

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// RegisterPublic adds routes that work without a session.
func (h *Handler) RegisterPublic(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.login)
}

// Register adds routes that need a session (mount behind RequireUser).
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/me", h.me)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
}

// RequireUser rejects requests without a valid "Authorization: Bearer" session
// token and puts the user's ID in the request context.
func (h *Handler) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.svc.Authenticate(bearer(r))
		if !ok {
			httpx.Error(w, http.StatusUnauthorized, "sign in required")
			return
		}
		next.ServeHTTP(w, r.WithContext(identity.WithUserID(r.Context(), u.ID)))
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	token, u, err := h.svc.Login(in.Email, in.Password)
	var locked *LockedError
	if errors.As(err, &locked) {
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(locked.RetryAfter.Seconds()))))
		httpx.Error(w, http.StatusTooManyRequests, err.Error())
		return
	}
	if errors.Is(err, ErrInvalidCredentials) {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not sign in")
		return
	}
	httpx.JSON(w, http.StatusOK, loginResponse{Token: token, User: u})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u, ok := h.svc.Authenticate(bearer(r))
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "sign in required")
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.svc.Logout(bearer(r))
	w.WriteHeader(http.StatusNoContent)
}

func bearer(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}
