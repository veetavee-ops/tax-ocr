package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"tax-ocr/backend/internal/db"
)

var jwtSecret = func() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "tax-ocr-dev-secret-change-in-production"
	}
	return []byte(s)
}()

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Type     string `json:"type"`
	jwt.RegisteredClaims
}

func issueToken(user db.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Name:     user.Name,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func issueRefreshToken(user db.User) (string, error) {
	claims := RefreshClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func parseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func (s *server) setup(w http.ResponseWriter, r *http.Request) {
	hasUsers, err := s.store.HasAnyUser(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if hasUsers {
		writeError(w, http.StatusForbidden, errors.New("ระบบมี user แล้ว ใช้ login แทน"))
		return
	}

	var req struct {
		TenantName string `json:"tenant_name"`
		TaxID      string `json:"tax_id"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tenant, err := s.store.CreateTenant(r.Context(), req.TenantName, req.TaxID, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	user, err := s.store.CreateUser(r.Context(), db.User{
		TenantID:     tenant.ID,
		Name:         req.Name,
		Email:        req.Email,
		Role:         "admin",
		PasswordHash: string(hashed),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	token, err := issueToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	refreshToken, err := issueRefreshToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
		"user":          map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "role": user.Role, "tenant_id": user.TenantID},
		"tenant":        tenant,
	})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if !loginLimiter.allow(extractIP(r)) {
		writeError(w, http.StatusTooManyRequests, errors.New("too many login attempts, please try again later"))
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, errors.New("email หรือ password ไม่ถูกต้อง"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, errors.New("email หรือ password ไม่ถูกต้อง"))
		return
	}

	token, err := issueToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	go func(u db.User, ip, ua string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := s.store.CreateAuditLog(ctx, db.AuditLog{
			TenantID:   u.TenantID,
			UserID:     u.ID,
			Action:     "login",
			EntityType: "auth",
			IPAddress:  ip,
			DeviceInfo: ua,
		}); err != nil {
			log.Printf("[audit] login write failed: %v", err)
		}
	}(user, extractIP(r), r.UserAgent())

	refreshToken, err := issueRefreshToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
		"user": map[string]any{
			"id":        user.ID,
			"name":      user.Name,
			"email":     user.Email,
			"role":      user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

func (s *server) lineLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LineUserID string `json:"line_user_id"`
		Name       string `json:"name"`
		TenantID   string `json:"tenant_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.LineUserID == "" || req.TenantID == "" {
		writeError(w, http.StatusBadRequest, errors.New("line_user_id and tenant_id are required"))
		return
	}
	if req.Name == "" {
		req.Name = "LINE User"
	}

	user, err := s.store.GetOrCreateLiffUser(r.Context(), req.LineUserID, req.Name, req.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	token, err := issueToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	refreshToken, err := issueRefreshToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":         token,
		"refresh_token": refreshToken,
		"user": map[string]any{
			"id":        user.ID,
			"name":      user.Name,
			"role":      user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

func (s *server) logout(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"message": "logged out"})
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        claims.UserID,
		"tenant_id": claims.TenantID,
		"name":      claims.Name,
		"role":      claims.Role,
	})
}

func (s *server) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	token, err := jwt.ParseWithClaims(req.RefreshToken, &RefreshClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		writeError(w, http.StatusUnauthorized, errors.New("invalid refresh token"))
		return
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || claims.Type != "refresh" {
		writeError(w, http.StatusUnauthorized, errors.New("invalid refresh token"))
		return
	}

	user, err := s.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, errors.New("user not found or inactive"))
		return
	}

	newToken, err := issueToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": newToken})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		claims, err := parseToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		r = r.WithContext(contextWithClaims(r.Context(), claims))
		next.ServeHTTP(w, r)
	})
}
