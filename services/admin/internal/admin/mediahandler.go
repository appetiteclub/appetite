package admin

import (
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) ListMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.ListMedia")
	defer finish()
	log := h.log(r)

	ctx := r.Context()

	items, err := h.service.ListMedia(ctx, MediaListParams{})
	if err != nil {
		log.Error("error listing media", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Show latest first for easier review
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	mediaCategories, mediaKinds, mediaTags := h.mediaLabelMaps(ctx)
	enabledCount := 0
	resources := make(map[string]int)
	for _, asset := range items {
		if asset.Enabled {
			enabledCount++
		}
		key := strings.ToLower(strings.TrimSpace(asset.TargetType))
		if key == "" {
			key = "unknown"
		}
		resources[key]++
	}

	tmpl, err := h.tmplMgr.Get("list-media.html")
	if err != nil {
		log.Error("error getting media list template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":                "Media Library",
		"ActiveNav":            "media",
		"Template":             "list-media-content",
		"MediaItems":           items,
		"EnabledCount":         enabledCount,
		"DisabledCount":        len(items) - enabledCount,
		"ResourceTypeCounters": resources,
		"MediaCategoryLabels":  mediaCategories,
		"MediaKindLabels":      mediaKinds,
		"MediaTagLabels":       mediaTags,
	}

	if err := tmpl.ExecuteTemplate(w, "list-media.html", data); err != nil {
		log.Error("error executing media list template", "error", err)
	}
}

func (h *Handler) ShowMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.ShowMedia")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	mediaID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	media, err := h.service.GetMedia(ctx, mediaID)
	if err != nil {
		log.Error("error fetching media", "error", err, "id", mediaID)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	mediaCategories, mediaKinds, mediaTags := h.mediaLabelMaps(ctx)
	mediaOptions := h.mediaFormOptions(ctx)
	tagSelected := make(map[string]struct{}, len(media.Tags))
	for _, tag := range media.Tags {
		tagSelected[tag.String()] = struct{}{}
	}

	tmpl, err := h.tmplMgr.Get("show-media.html")
	if err != nil {
		log.Error("error getting media show template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":               "Media Details",
		"ActiveNav":           "media",
		"Template":            "show-media",
		"Media":               media,
		"MediaCategoryLabels": mediaCategories,
		"MediaKindLabels":     mediaKinds,
		"MediaTagLabels":      mediaTags,
		"MediaCategories":     mediaOptions.Categories,
		"MediaKinds":          mediaOptions.Kinds,
		"MediaTagGroups":      mediaOptions.TagGroups,
		"MediaTagSelected":    tagSelected,
	}

	if err := tmpl.ExecuteTemplate(w, "show-media.html", data); err != nil {
		log.Error("error executing media show template", "error", err)
	}
}

func (h *Handler) UpdateMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.UpdateMedia")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	mediaID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	update := UpdateMediaRequest{}

	if kind := strings.TrimSpace(r.FormValue("media_kind")); kind != "" {
		update.Kind = &kind
	}

	if category := strings.TrimSpace(r.FormValue("media_category_id")); category != "" {
		categoryID, err := uuid.Parse(category)
		if err != nil {
			http.Error(w, "Invalid category", http.StatusBadRequest)
			return
		}
		update.CategoryID = &categoryID
	}

	enabled := r.FormValue("media_enabled") == "1"
	update.Enabled = &enabled

	tags := gatherMediaTagIDsFromValues(map[string][]string(r.PostForm), DefaultMediaTagSetNames()...)
	tagsCopy := append([]uuid.UUID{}, tags...)
	if tagsCopy == nil {
		tagsCopy = []uuid.UUID{}
	}
	update.Tags = &tagsCopy

	if storagePath := strings.TrimSpace(r.FormValue("media_storage_path")); storagePath != "" {
		update.StoragePath = &storagePath
	}

	if _, err := h.service.UpdateMedia(ctx, mediaID, update); err != nil {
		log.Error("error updating media", "error", err, "id", mediaID)
		http.Error(w, "Could not update media", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/show-media/"+mediaID.String(), http.StatusSeeOther)
}

func (h *Handler) EnableMedia(w http.ResponseWriter, r *http.Request) {
	h.toggleMediaEnabled(w, r, true)
}

func (h *Handler) DisableMedia(w http.ResponseWriter, r *http.Request) {
	h.toggleMediaEnabled(w, r, false)
}

func (h *Handler) toggleMediaEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	w, r, finish := h.http.Start(w, r, "Handler.ToggleMediaEnabled")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	mediaID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if enabled {
		if _, err := h.service.EnableMedia(ctx, mediaID); err != nil {
			log.Error("error enabling media", "error", err, "id", mediaID)
			http.Error(w, "Could not enable media", http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := h.service.DisableMedia(ctx, mediaID); err != nil {
			log.Error("error disabling media", "error", err, "id", mediaID)
			http.Error(w, "Could not disable media", http.StatusInternalServerError)
			return
		}
	}

	target := r.Referer()
	if target == "" {
		target = "/show-media/" + mediaID.String()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.DeleteMedia")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	mediaID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteMedia(ctx, mediaID); err != nil {
		log.Error("error deleting media", "error", err, "id", mediaID)
		http.Error(w, "Could not delete media", http.StatusInternalServerError)
		return
	}

	target := r.Referer()
	if target == "" || strings.Contains(target, "/show-media/") {
		target = "/list-media"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
