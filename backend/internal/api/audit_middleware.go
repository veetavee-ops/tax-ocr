package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"tax-ocr/backend/internal/db"
)

type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

func auditMiddleware(store *db.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			rc := &responseCapture{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rc, r)

			if rc.statusCode < 200 || rc.statusCode >= 300 {
				return
			}
			claims := claimsFromContext(r.Context())
			if claims == nil {
				return
			}

			action, entityType, entityID := parseAuditInfo(r)
			ip := extractIP(r)
			ua := r.UserAgent()
			tenantID := claims.TenantID
			userID := claims.UserID

			// Use background context so the goroutine isn't cancelled when request ends
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := store.CreateAuditLog(ctx, db.AuditLog{
					TenantID:   tenantID,
					UserID:     userID,
					Action:     action,
					EntityType: entityType,
					EntityID:   entityID,
					IPAddress:  ip,
					DeviceInfo: ua,
				}); err != nil {
					log.Printf("[audit] write failed: %v", err)
				}
			}()
		})
	}
}

// parseAuditInfo derives action, entity_type and entity_id from the request.
func parseAuditInfo(r *http.Request) (action, entityType, entityID string) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	segments := strings.Split(strings.Trim(path, "/"), "/")

	resource := ""
	if len(segments) > 0 {
		resource = segments[0]
	}

	// Special sub-action keywords override the HTTP-method default
	last := segments[len(segments)-1]
	switch last {
	case "upload":
		action = "upload"
	case "import":
		action = "import"
	case "export":
		action = "export"
	case "resolve":
		action = "resolve"
	case "reject":
		action = "reject"
	case "accept":
		action = "accept"
	case "complete":
		action = "complete"
	case "restore":
		action = "restore"
	case "payout":
		action = "payout"
	case "test":
		action = "test"
	case "verify":
		action = "verify"
	default:
		switch r.Method {
		case http.MethodPost:
			action = "create"
		case http.MethodPut, http.MethodPatch:
			action = "update"
		case http.MethodDelete:
			action = "delete"
		default:
			action = strings.ToLower(r.Method)
		}
	}

	entityType = resourceToEntityType(resource)

	// entity_id is the segment after the resource if it looks like a UUID
	if len(segments) > 1 && looksLikeID(segments[1]) {
		entityID = segments[1]
	}

	return action, entityType, entityID
}

func resourceToEntityType(resource string) string {
	m := map[string]string{
		"invoices":         "invoice",
		"documents":        "document_import",
		"users":            "user",
		"tenants":          "tenant",
		"branches":         "branch",
		"rules":            "classification_rule",
		"hitl":             "hitl_queue",
		"reviewers":        "reviewer",
		"reviewer-tasks":   "reviewer_task",
		"conversations":    "conversation",
		"messages":         "message",
		"archive":          "archive",
		"storage":          "tenant_storage_config",
		"reward":           "reward_config",
		"ocr":              "ocr_config",
		"auth":             "auth",
	}
	if v, ok := m[resource]; ok {
		return v
	}
	return resource
}

func looksLikeID(s string) bool {
	if len(s) != 36 {
		return false
	}
	dashes := 0
	for _, c := range s {
		if c == '-' {
			dashes++
		}
	}
	return dashes == 4
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.SplitN(xff, ",", 2)[0]
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}
