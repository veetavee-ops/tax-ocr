package api

import (
	"errors"
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

func issueToken(user db.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Name:     user.Name,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
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

	tenant, err := s.store.CreateTenant(r.Context(), req.TenantName, req.TaxID)
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

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":  token,
		"user":   map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "role": user.Role, "tenant_id": user.TenantID},
		"tenant": tenant,
	})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":        user.ID,
			"name":      user.Name,
			"email":     user.Email,
			"role":      user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":   claims.UserID,
		"tenant_id": claims.TenantID,
		"name":      claims.Name,
		"role":      claims.Role,
	})
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
