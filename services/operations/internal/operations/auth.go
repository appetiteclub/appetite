package operations

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ShowSignIn displays the sign-in page
func (h *Handler) ShowSignIn(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.ShowSignIn")
	defer finish()

	data := map[string]interface{}{
		"Title":    "Sign In - Operations",
		"Template": "signin",
		"HideNav":  true,
	}

	h.renderTemplate(w, "signin.html", "base.html", data)
}

// HandleSignIn processes sign-in requests
func (h *Handler) HandleSignIn(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.HandleSignIn")
	defer finish()

	renderError := func(message string) {
		data := map[string]interface{}{
			"Title":    "Sign In - Operations",
			"Template": "signin",
			"HideNav":  true,
			"Error":    message,
		}
		h.renderTemplate(w, "signin.html", "base.html", data)
	}

	if err := r.ParseForm(); err != nil {
		h.log().Debug("failed to parse form", "error", err)
		renderError("Failed to parse form. Please try again.")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.log().Debug("missing email or password")
		renderError("Email and password are required.")
		return
	}

	// Authenticate with AuthN service
	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	authResp, err := h.authnClient.Request(r.Context(), http.MethodPost, "/authn/signin", payload)
	if err != nil {
		h.log().Debug("authentication failed", "error", err)
		renderError("Invalid email or password. Please try again.")
		return
	}

	if authResp == nil || authResp.Data == nil {
		h.log().Error("authn signin returned empty response")
		renderError("Authentication service unavailable. Please try again later.")
		return
	}

	// Extract user data from response
	responsePayload, ok := authResp.Data.(map[string]interface{})
	if !ok {
		h.log().Error("unexpected signin response type")
		renderError("Authentication error. Please try again.")
		return
	}

	userRaw, ok := responsePayload["user"]
	if !ok {
		h.log().Error("signin response missing user field")
		renderError("Authentication error. Please try again.")
		return
	}

	userData, ok := userRaw.(map[string]interface{})
	if !ok {
		h.log().Error("unexpected user data type")
		renderError("Authentication error. Please try again.")
		return
	}

	userID, _ := userData["id"].(string)
	username, _ := userData["username"].(string)
	name, _ := userData["name"].(string)
	userEmail, _ := userData["email"].(string)

	// Create session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Username:  username,
		Name:      name,
		Email:     userEmail,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(h.sessionStore.ttl),
	}

	if err := h.sessionStore.Save(session); err != nil {
		h.log().Error("failed to save session", "error", err)
		renderError("Session error. Please try again.")
		return
	}

	// Set session cookie
	sessionName, _ := h.config.GetString("auth.session.name")
	http.SetCookie(w, &http.Cookie{
		Name:     sessionName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionStore.ttl.Seconds()),
	})

	// Redirect to home
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// HandleSignOut processes sign-out requests
func (h *Handler) HandleSignOut(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.HandleSignOut")
	defer finish()

	sessionName, _ := h.config.GetString("auth.session.name")
	cookie, err := r.Cookie(sessionName)
	if err == nil && cookie.Value != "" {
		h.sessionStore.Delete(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Redirect to signin
	w.Header().Set("HX-Redirect", "/signin")
	w.WriteHeader(http.StatusOK)
}

// SessionMiddleware validates session for protected routes
func (h *Handler) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionName, _ := h.config.GetString("auth.session.name")
		cookie, err := r.Cookie(sessionName)
		if err != nil {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}

		session, err := h.sessionStore.Get(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}

		// Add session to context
		ctx := context.WithValue(r.Context(), "session", session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
