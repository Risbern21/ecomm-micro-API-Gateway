// Package handler provides HandlerFunc
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/risbern21/api_gateway/internal/token"
	"github.com/risbern21/api_gateway/model"
	"github.com/risbern21/api_gateway/util"
)

type Handler struct {
	tokenMaker *token.JWTMaker
}

func NewHandler(secretKey string) *Handler {
	return &Handler{
		tokenMaker: token.NewJWTMaker(secretKey),
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	u := model.NewUser()

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, fmt.Sprintf("error creating user %v", err), http.StatusBadRequest)
		return
	}

	password, err := util.HashPassword(u.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf("error creating user %v", err), http.StatusInternalServerError)
		return
	}

	u.Password = password

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := u.AddUser(ctx); err != nil {
		http.Error(w, fmt.Sprintf("error creating user %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&u)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req := &LoginReq{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding user %v", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	u := model.NewUser()
	err := u.GetUserByEmail(ctx, req.Email)
	if err != nil {
		http.Error(w, fmt.Sprintf("error while fetching user from db %v", err), http.StatusInternalServerError)
		return
	}

	if err := util.CheckPassword(req.Password, u.Password); err != nil {
		http.Error(w, "invalid password", http.StatusBadRequest)
		return
	}

	//generate JWT token and return
	accessToken, accessTokenClaims, err := h.tokenMaker.CreateToken(u.ID, u.Email, u.Role.String(), 15*time.Minute)
	if err != nil {
		http.Error(w, fmt.Sprintf("error while creating token : %v", err), http.StatusInternalServerError)
		return
	}

	refreshToken, refreshTokenClaims, err := h.tokenMaker.CreateToken(u.ID, u.Email, u.Role.String(), 24*time.Hour)
	if err != nil {
		http.Error(w, fmt.Sprintf("error while creating token : %v", err), http.StatusInternalServerError)
		return
	}

	s := model.NewSession()
	s.ID = refreshTokenClaims.RegisteredClaims.ID
	s.UserEmail = u.Email
	s.RefreshToken = refreshToken
	s.IsRevoked = false
	s.ExpiresAt = refreshTokenClaims.ExpiresAt.Time
	if err := s.CreateSession(); err != nil {
		http.Error(w, fmt.Sprintf("unable to create session : %v", err), http.StatusInternalServerError)
		return
	}

	res := &LoginRes{
		SessionID:             s.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenClaims.ExpiresAt.Time,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshTokenClaims.ExpiresAt.Time,
		User:                  *u,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query["id"][0]
	if id == "" {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	s := model.NewSession()
	s.ID = id
	if err := s.DeleteSession(); err != nil {
		http.Error(w, fmt.Sprintf("unable to create session : %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RenewAccessToken(w http.ResponseWriter, r *http.Request) {
	req := &RenewAccessTokenReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "unable to decode req body", http.StatusBadRequest)
		return
	}

	refreshTokenClaims, err := h.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("error while verifying token : %v", err), http.StatusUnauthorized)
		return
	}

	s := model.NewSession()
	s.ID = refreshTokenClaims.RegisteredClaims.ID
	if err := s.GetSession(); err != nil {
		http.Error(w, fmt.Sprintf("error while fetching session : %v", err), http.StatusInternalServerError)
		return
	}

	if s.IsRevoked {
		http.Error(w, "session is revoked", http.StatusUnauthorized)
		return
	}

	if s.UserEmail != refreshTokenClaims.Email {
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	accessToken, accessTokenClaims, err := h.tokenMaker.CreateToken(refreshTokenClaims.ID, refreshTokenClaims.Email, refreshTokenClaims.Role, 15*time.Minute)
	if err != nil {
		http.Error(w, "error creating access token", http.StatusInternalServerError)
		return
	}

	res := &RenewAccessTokenRes{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenClaims.ExpiresAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query["id"][0]
	if id == "" {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	s := model.NewSession()
	s.ID = id
	if err := s.RevokeSession(); err != nil {
		http.Error(w, "unable to revoke session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
