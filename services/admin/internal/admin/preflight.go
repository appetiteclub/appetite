package admin

import (
	"errors"
	"net/http"
	"strings"
)

type ActorCtxKey string

const actorKey = "actor"

type Actor struct {
	ID     string
	Grants []string
}

func (h *Handler) GetActorFromContext(r *http.Request) Actor {
	actor, ok := r.Context().Value(actorKey).(Actor)
	if !ok {
		return Actor{}
	}
	return actor
}

// pfc returns (0, nil) if allowed; otherwise (httpStatus, error) to send back.
func (h *Handler) pfc(r *http.Request, permission, resourceID string) (int, error) {
	actor := h.GetActorFromContext(r)
	if actor.ID == "" {
		return http.StatusUnauthorized, errors.New(http.StatusText(http.StatusUnauthorized))
	}

	perm := strings.TrimSpace(strings.ToLower(permission))
	if perm == "" {
		return http.StatusForbidden, errors.New(http.StatusText(http.StatusForbidden))
	}

	allowed, err := h.authZClt.CheckPermission(r.Context(), actor.ID, perm, resourceID)
	if err != nil {
		h.log(r).Error("authz check failed", "error", err, "perm", perm, "resource", resourceID)
		return http.StatusInternalServerError, errors.New(http.StatusText(http.StatusInternalServerError))
	}
	if !allowed {
		return http.StatusForbidden, errors.New(http.StatusText(http.StatusForbidden))
	}
	return 0, nil
}
