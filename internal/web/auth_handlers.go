package web

import (
	"encoding/json"
	"net/http"

	"github.com/flow-agent/flow-agent/internal/auth"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
)

type sendSMSRequest struct {
	Phone string `json:"phone"`
}

type loginSMSRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type authStatusResponse struct {
	AuthEnabled bool `json:"auth_enabled"`
	LoggedIn    bool `json:"logged_in"`
}

type loginResponse struct {
	Token string     `json:"token"`
	User  *auth.User `json:"user"`
}

func (h *apiHandler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	resp := authStatusResponse{AuthEnabled: h.authCfg.Enabled}
	if h.authCfg.Enabled {
		if _, err := auth.UserFromRequest(r, h.authCfg.JWTSecret); err == nil {
			resp.LoggedIn = true
		}
	}
	writeJSON(w, resp)
}

func (h *apiHandler) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if !h.authCfg.Enabled {
		writeJSON(w, map[string]any{"auth_enabled": false})
		return
	}
	claims, err := auth.UserFromRequest(r, h.authCfg.JWTSecret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := h.authStore.GetUser(claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}
	writeJSON(w, user)
}

func (h *apiHandler) handleAuthSendSMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if h.authSvc == nil {
		http.Error(w, "auth not configured", http.StatusServiceUnavailable)
		return
	}
	if h.smsSendLimit != nil && !h.smsSendLimit.Allow(auth.ClientIP(r)) {
		http.Error(w, auth.ErrRateLimited.Error(), http.StatusTooManyRequests)
		return
	}
	var req sendSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.authSvc.SendSMSCode(r.Context(), req.Phone); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "sent"})
}

func (h *apiHandler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if h.authSvc == nil {
		http.Error(w, "auth not configured", http.StatusServiceUnavailable)
		return
	}
	if h.smsLoginLimit != nil && !h.smsLoginLimit.Allow(auth.ClientIP(r)) {
		http.Error(w, auth.ErrRateLimited.Error(), http.StatusTooManyRequests)
		return
	}
	var req loginSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	token, user, err := h.authSvc.LoginWithSMS(r.Context(), req.Phone, req.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "flowagent_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})
	writeJSON(w, loginResponse{Token: token, User: user})
}

func (h *apiHandler) handleListRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	claims, err := auth.RequireUser(r, h.authCfg)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := ""
	if claims != nil {
		userID = claims.UserID
	}
	app, err := config.Load(h.root, "micro-movie-wan-flash")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	store := runctx.NewStore(app.RunsDir)
	list, err := store.ListRuns(userID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"runs": list})
}

func (h *apiHandler) currentUserID(r *http.Request) (string, error) {
	claims, err := auth.RequireUser(r, h.authCfg)
	if err != nil {
		return "", err
	}
	if claims == nil {
		return "", nil
	}
	return claims.UserID, nil
}
