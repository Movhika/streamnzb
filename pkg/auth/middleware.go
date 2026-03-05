package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

func DeviceFromContext(r *http.Request) (*Device, bool) {
	device, ok := r.Context().Value(userContextKey).(*Device)
	return device, ok
}

func ContextWithDevice(ctx context.Context, device *Device) context.Context {
	return context.WithValue(ctx, userContextKey, device)
}

func AuthMiddleware(deviceManager *DeviceManager, getAdminUsername, getAdminToken func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			adminUsername := ""
			if getAdminUsername != nil {
				adminUsername = getAdminUsername()
			}
			adminToken := ""
			if getAdminToken != nil {
				adminToken = getAdminToken()
			}
			var device *Device
			var err error

			cookie, err := r.Cookie("auth_session")
			if err == nil && cookie != nil {
				device, err = deviceManager.AuthenticateToken(cookie.Value, adminUsername, adminToken)
				if err == nil {
					ctx := context.WithValue(r.Context(), userContextKey, device)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Cookie present but invalid (e.g. after container restart with a new token).
				// Clear it so the browser doesn't keep sending a stale credential.
				http.SetCookie(w, &http.Cookie{
					Name:     "auth_session",
					Value:    "",
					Path:     "/",
					HttpOnly: true,
					MaxAge:   -1,
				})
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					token := parts[1]
					device, err = deviceManager.AuthenticateToken(token, adminUsername, adminToken)
					if err == nil {
						ctx := context.WithValue(r.Context(), userContextKey, device)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			token := r.URL.Query().Get("token")
			if token != "" {
				device, err = deviceManager.AuthenticateToken(token, adminUsername, adminToken)
				if err == nil {
					ctx := context.WithValue(r.Context(), userContextKey, device)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized",
			})
		})
	}
}
