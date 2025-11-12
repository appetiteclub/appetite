package admin

import (
	"context"
	"net/http"
	"net/url"
)

const (
	sessionCookieName = "session_id"
	signInURL         = "/signin"
)

type contextKey string

const (
	UserIDContextKey contextKey = "userID"
)

// SessionMiddleware handles session validation and user context injection.
// It doesn't deal with permissions (AuthZ), only authentication (AuthN).
func SessionMiddleware(sessionValidator func(sessionID string) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				redirectToLogin(w, r)
				return
			}

			userID, err := sessionValidator(cookie.Value)
			if err != nil || userID == "" {
				clearSessionCookie(w)
				redirectToLogin(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			actor := Actor{
				ID:     userID,
				Grants: []string{"*:*"},
			}
			ctx = context.WithValue(ctx, actorKey, actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	originalURL := r.URL.RequestURI()
	target := signInURL + "?next=" + url.QueryEscape(originalURL)
	http.Redirect(w, r, target, http.StatusFound)
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// GetUserID extracts the user ID from the request context.
func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDContextKey).(string)
	return id, ok
}
