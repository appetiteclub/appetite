package media

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxBodyBytes = 5 << 20 // 5MB

type Handler struct {
	service *Service
	deps    *apt.Deps
	tlm     *telemetry.HTTP
}

func NewHandler(service *Service, deps *apt.Deps) *Handler {
	return &Handler{
		service: service,
		deps:    deps,
		tlm: telemetry.NewHTTP(
			telemetry.WithTracer(deps.Tracer),
			telemetry.WithMetrics(deps.Metrics),
		),
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/media", func(r chi.Router) {
		r.Post("/", h.createMedia)
		r.Get("/", h.listMedia)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.getMedia)
			r.Put("/", h.updateMedia)
			r.Delete("/", h.deleteMedia)
			r.Post("/enable", h.enableMedia)
			r.Post("/disable", h.disableMedia)
		})
	})
}

type resolutionPayload struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type variantPayload struct {
	Path     string `json:"path"`
	MimeType string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Filesize int64  `json:"filesize"`
}

type createRequest struct {
	ResourceType string                    `json:"resource_type"`
	ResourceID   string                    `json:"resource_id"`
	MimeType     string                    `json:"mime_type"`
	Resolution   resolutionPayload         `json:"resolution"`
	Filesize     int64                     `json:"filesize"`
	Enabled      *bool                     `json:"enabled"`
	Kind         string                    `json:"kind"`
	CategoryID   string                    `json:"category_id"`
	Tags         []string                  `json:"tags"`
	Metadata     map[string]any            `json:"metadata"`
	StoragePath  string                    `json:"storage_path"`
	Variants     map[string]variantPayload `json:"variants"`
}

type updateRequest struct {
	MimeType    *string                    `json:"mime_type"`
	Resolution  *resolutionPayload         `json:"resolution"`
	Filesize    *int64                     `json:"filesize"`
	Enabled     *bool                      `json:"enabled"`
	Kind        *string                    `json:"kind"`
	CategoryID  *string                    `json:"category_id"`
	Tags        *[]string                  `json:"tags"`
	Metadata    *map[string]any            `json:"metadata"`
	StoragePath *string                    `json:"storage_path"`
	Variants    *map[string]variantPayload `json:"variants"`
}

func (h *Handler) createMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateMedia")
	defer finish()

	log := h.log(r)
	ctx := r.Context()
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.createMediaMultipart(w, r, log)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var payload createRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Debug("invalid payload", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	resourceID, err := uuid.Parse(payload.ResourceID)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid resource_id")
		return
	}
	categoryID, err := uuid.Parse(payload.CategoryID)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid category_id")
		return
	}
	tags, err := parseUUIDList(payload.Tags)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid tag value")
		return
	}

	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}

	variants := make(map[string]Variant)
	for name, v := range payload.Variants {
		variants[name] = Variant{
			Path:     v.Path,
			MimeType: v.MimeType,
			Width:    v.Width,
			Height:   v.Height,
			Filesize: v.Filesize,
		}
	}

	input := CreateInput{
		ResourceType: payload.ResourceType,
		ResourceID:   resourceID,
		MimeType:     payload.MimeType,
		Resolution:   Resolution{Width: payload.Resolution.Width, Height: payload.Resolution.Height},
		Filesize:     payload.Filesize,
		Enabled:      enabled,
		Kind:         Kind(strings.TrimSpace(payload.Kind)),
		CategoryID:   categoryID,
		Tags:         tags,
		Metadata:     payload.Metadata,
		StoragePath:  payload.StoragePath,
		Variants:     variants,
	}

	media, err := h.service.CreateMedia(ctx, input)
	if err != nil {
		log.Error("cannot create media", "error", err)
		apt.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondCreated(w, media)
}

func (h *Handler) createMediaMultipart(w http.ResponseWriter, r *http.Request, log apt.Logger) {
	if err := r.ParseMultipartForm(maxBodyBytes * 4); err != nil {
		log.Debug("invalid multipart payload", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid multipart payload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "File is required")
		return
	}
	defer file.Close()

	buffer := &bytes.Buffer{}
	if _, err := io.Copy(buffer, file); err != nil {
		log.Error("cannot read uploaded file", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	if buffer.Len() == 0 {
		apt.RespondError(w, http.StatusBadRequest, "Uploaded file is empty")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(buffer.Bytes())
	}

	resourceType := strings.TrimSpace(r.FormValue("resource_type"))
	if resourceType == "" {
		resourceType = "property"
	}
	resourceID, err := uuid.Parse(strings.TrimSpace(r.FormValue("resource_id")))
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid resource_id")
		return
	}
	categoryID, err := uuid.Parse(strings.TrimSpace(r.FormValue("category_id")))
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid category_id")
		return
	}

	width := parseIntField(r.FormValue("resolution_width"))
	height := parseIntField(r.FormValue("resolution_height"))
	if width <= 0 || height <= 0 {
		cfg, format, err := image.DecodeConfig(bytes.NewReader(buffer.Bytes()))
		if err == nil {
			width = cfg.Width
			height = cfg.Height
			if mimeType == "" {
				mimeType = "image/" + format
			}
		} else {
			log.Error("cannot determine image dimensions", "error", err)
			apt.RespondError(w, http.StatusBadRequest, "Image dimensions could not be determined")
			return
		}
	}

	tags, _ := parseUUIDList(r.MultipartForm.Value["tags"])
	for _, field := range []string{"media_usage", "media_style", "media_mood", "media_capture_conditions", "media_feature_tags"} {
		vals := r.MultipartForm.Value[field]
		if len(vals) == 0 {
			continue
		}
		parsed, err := parseUUIDList(vals)
		if err == nil {
			tags = append(tags, parsed...)
		}
	}
	if len(tags) > 0 {
		tags = uniqUUIDs(tags)
	}

	enabled := true
	if flag := strings.TrimSpace(r.FormValue("enabled")); flag != "" {
		if b, err := strconv.ParseBool(flag); err == nil {
			enabled = b
		}
	}

	metadata := map[string]any{}
	if meta := strings.TrimSpace(r.FormValue("metadata")); meta != "" {
		var tmp map[string]any
		if err := json.Unmarshal([]byte(meta), &tmp); err == nil {
			metadata = tmp
		}
	}
	metadata["original_name"] = header.Filename

	kind := strings.TrimSpace(r.FormValue("kind"))
	if kind == "" {
		kind = "real"
	}

	input := CreateInput{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		MimeType:     mimeType,
		Resolution:   Resolution{Width: width, Height: height},
		Filesize:     int64(buffer.Len()),
		Enabled:      enabled,
		Kind:         Kind(kind),
		CategoryID:   categoryID,
		Tags:         tags,
		Metadata:     metadata,
		Data:         bytes.NewReader(buffer.Bytes()),
	}

	media, err := h.service.CreateMedia(r.Context(), input)
	if err != nil {
		log.Error("cannot create media", "error", err)
		apt.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondCreated(w, media)
}

func (h *Handler) getMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetMedia")
	defer finish()
	log := h.log(r)

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	media, err := h.service.GetMedia(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apt.RespondError(w, http.StatusNotFound, "Media not found")
			return
		}
		log.Error("cannot load media", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not load media")
		return
	}

	links := apt.RESTfulLinksFor(media)
	apt.RespondSuccess(w, media, links...)
}

func (h *Handler) listMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListMedia")
	defer finish()

	resourceType := strings.TrimSpace(r.URL.Query().Get("resource_type"))
	resourceIDStr := strings.TrimSpace(r.URL.Query().Get("resource_id"))
	var resourceID uuid.UUID
	if resourceIDStr != "" {
		id, err := uuid.Parse(resourceIDStr)
		if err != nil {
			apt.RespondError(w, http.StatusBadRequest, "Invalid resource_id")
			return
		}
		resourceID = id
	}

	items, err := h.service.ListMedia(r.Context(), ListFilter{ResourceType: resourceType, ResourceID: resourceID})
	if err != nil {
		h.log(r).Error("cannot list media", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not list media")
		return
	}

	apt.RespondSuccess(w, items)
}

func (h *Handler) updateMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateMedia")
	defer finish()
	log := h.log(r)

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var payload updateRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Debug("invalid payload", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	var res *Resolution
	if payload.Resolution != nil {
		res = &Resolution{Width: payload.Resolution.Width, Height: payload.Resolution.Height}
	}
	var kind *Kind
	if payload.Kind != nil {
		k := Kind(strings.TrimSpace(*payload.Kind))
		kind = &k
	}
	var categoryID *uuid.UUID
	if payload.CategoryID != nil {
		id, err := uuid.Parse(*payload.CategoryID)
		if err != nil {
			apt.RespondError(w, http.StatusBadRequest, "Invalid category_id")
			return
		}
		categoryID = &id
	}
	var tags *[]uuid.UUID
	if payload.Tags != nil {
		parsed, err := parseUUIDList(*payload.Tags)
		if err != nil {
			apt.RespondError(w, http.StatusBadRequest, "Invalid tag value")
			return
		}
		tags = &parsed
	}
	var variants *map[string]Variant
	if payload.Variants != nil {
		converted := make(map[string]Variant, len(*payload.Variants))
		for name, v := range *payload.Variants {
			converted[name] = Variant{
				Path:     v.Path,
				MimeType: v.MimeType,
				Width:    v.Width,
				Height:   v.Height,
				Filesize: v.Filesize,
			}
		}
		variants = &converted
	}

	input := UpdateInput{
		ID:          id,
		MimeType:    payload.MimeType,
		Resolution:  res,
		Filesize:    payload.Filesize,
		Enabled:     payload.Enabled,
		Kind:        kind,
		CategoryID:  categoryID,
		Tags:        tags,
		Metadata:    payload.Metadata,
		StoragePath: payload.StoragePath,
		Variants:    variants,
	}

	media, err := h.service.UpdateMedia(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apt.RespondError(w, http.StatusNotFound, "Media not found")
			return
		}
		log.Error("cannot update media", "error", err)
		apt.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	links := apt.RESTfulLinksFor(media)
	apt.RespondSuccess(w, media, links...)
}

func (h *Handler) deleteMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteMedia")
	defer finish()
	log := h.log(r)

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			apt.RespondError(w, http.StatusNotFound, "Media not found")
			return
		}
		log.Error("cannot delete media", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not delete media")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) enableMedia(w http.ResponseWriter, r *http.Request) {
	h.toggleMedia(w, r, true)
}

func (h *Handler) disableMedia(w http.ResponseWriter, r *http.Request) {
	h.toggleMedia(w, r, false)
}

func (h *Handler) toggleMedia(w http.ResponseWriter, r *http.Request, enabled bool) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ToggleMedia")
	defer finish()
	log := h.log(r)

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	media, err := h.service.SetMediaEnabled(r.Context(), id, enabled)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apt.RespondError(w, http.StatusNotFound, "Media not found")
			return
		}
		log.Error("cannot toggle media", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not update media")
		return
	}

	apt.RespondSuccess(w, media, apt.RESTfulLinksFor(media)...)
}

func (h *Handler) log(r *http.Request) apt.Logger {
	logger := h.deps.Logger
	if logger == nil {
		return apt.NewNoopLogger()
	}
	if reqID, ok := r.Context().Value("request_id").(string); ok && reqID != "" {
		return logger.With("request_id", reqID)
	}
	return logger
}

func (h *Handler) parseIDParam(w http.ResponseWriter, r *http.Request, log apt.Logger) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		apt.RespondError(w, http.StatusBadRequest, "Missing id parameter")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid id parameter")
		return uuid.Nil, false
	}
	return id, true
}

func parseUUIDList(values []string) ([]uuid.UUID, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func parseIntField(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

func uniqUUIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func respondCreated(w http.ResponseWriter, media *Media) {
	resp := apt.SuccessResponse{
		Data:  media,
		Links: apt.RESTfulLinksFor(media),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
