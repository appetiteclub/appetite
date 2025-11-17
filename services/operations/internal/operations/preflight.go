package operations

import (
	"errors"
	"net/http"
	"strings"
)

func (h *Handler) requirePermission(w http.ResponseWriter, r *http.Request, permission string) bool {
	status, err := h.preflight(r, permission)
	if err == nil {
		return true
	}

	if status == http.StatusUnauthorized {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
	} else if status == http.StatusForbidden {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	} else {
		if status == 0 {
			status = http.StatusInternalServerError
		}
		http.Error(w, http.StatusText(status), status)
	}

	if err != nil {
		h.log().Error("preflight failed", "permission", permission, "status", status, "error", err)
	}

	return false
}

func (h *Handler) preflight(r *http.Request, permission string) (int, error) {
	perm := strings.TrimSpace(permission)
	if perm == "" {
		return http.StatusForbidden, errors.New("permission required")
	}

	session, _ := r.Context().Value("session").(*Session)
	if session == nil || session.UserID == "" {
		return http.StatusUnauthorized, errors.New("missing session")
	}

	if h.authzHelper == nil {
		return http.StatusInternalServerError, errors.New("authorization helper not configured")
	}

	allowed, err := h.authzHelper.CheckPermission(r.Context(), session.UserID, perm, "*")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !allowed {
		return http.StatusForbidden, errors.New("permission denied")
	}

	return 0, nil
}
