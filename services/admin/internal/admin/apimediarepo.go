package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

// APIMediaRepo implements MediaRepo using the media service HTTP API.
type APIMediaRepo struct {
	client     *apt.ServiceClient
	baseURL    string
	httpClient *http.Client
	logger     apt.Logger
}

// NewAPIMediaRepo creates a new media repository backed by the media service.
// Returns nil if services.media.url is not configured (media service is optional).
func NewAPIMediaRepo(config *apt.Config, logger apt.Logger) *APIMediaRepo {
	mediaURL, _ := config.GetString("services.media.url")
	if mediaURL == "" {
		return nil
	}

	client := apt.NewServiceClient(mediaURL)
	if client == nil {
		return nil
	}

	return &APIMediaRepo{
		client:     client,
		baseURL:    strings.TrimRight(mediaURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// List returns media assets using optional resource filters.
func (r *APIMediaRepo) List(ctx context.Context, params MediaListParams) ([]*Media, error) {
	if r == nil || r.client == nil {
		return []*Media{}, nil
	}

	path := "/media"
	q := url.Values{}
	if params.ResourceType != "" {
		q.Set("resource_type", params.ResourceType)
	}
	if params.ResourceID != nil && *params.ResourceID != uuid.Nil {
		q.Set("resource_id", params.ResourceID.String())
	}
	if encoded := q.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := r.client.Request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list media: %w", err)
	}

	var payload interface{}
	if resp != nil {
		payload = resp.Data
	}
	return parseMediaSlice(payload)
}

// ListByTarget returns all media assets associated with a given target.
func (r *APIMediaRepo) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]*Media, error) {
	if targetType == "" || targetID == uuid.Nil {
		return []*Media{}, nil
	}
	return r.List(ctx, MediaListParams{ResourceType: targetType, ResourceID: &targetID})
}

// Create uploads a new media asset to the media service.
func (r *APIMediaRepo) Create(ctx context.Context, req CreateMediaRequest) (*Media, error) {
	if r == nil || r.httpClient == nil || r.baseURL == "" {
		return nil, fmt.Errorf("media repository not configured")
	}

	if len(req.Data) == 0 {
		return nil, fmt.Errorf("media data cannot be empty")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writeField := func(key, value string) {
		if value == "" {
			return
		}
		_ = writer.WriteField(key, value)
	}

	writeField("resource_type", req.ResourceType)
	writeField("resource_id", req.ResourceID.String())
	writeField("category_id", req.CategoryID.String())
	writeField("kind", req.Kind)
	writeField("enabled", strconv.FormatBool(req.Enabled))
	writeField("mime_type", req.MimeType)
	writeField("filesize", strconv.FormatInt(req.Filesize, 10))
	if req.Resolution.Width > 0 {
		writeField("resolution_width", strconv.Itoa(req.Resolution.Width))
	}
	if req.Resolution.Height > 0 {
		writeField("resolution_height", strconv.Itoa(req.Resolution.Height))
	}

	if len(req.Metadata) > 0 {
		metaBytes, _ := json.Marshal(req.Metadata)
		writeField("metadata", string(metaBytes))
	}

	for _, tag := range req.Tags {
		writeField("tags", tag.String())
	}

	fileName := req.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("media-%s", req.ResourceID.String())
	}
	filePart, err := writer.CreateFormFile("file", path.Base(fileName))
	if err != nil {
		return nil, fmt.Errorf("cannot create multipart file: %w", err)
	}
	if _, err := filePart.Write(req.Data); err != nil {
		return nil, fmt.Errorf("cannot write file payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("cannot close multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/media", body)
	if err != nil {
		return nil, fmt.Errorf("cannot build media request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")
	if req.MimeType != "" {
		httpReq.Header.Set("X-File-Mime", req.MimeType)
	}
	if requestID := apt.RequestIDFrom(ctx); requestID != "" {
		httpReq.Header.Set(apt.RequestIDHeader, requestID)
	}

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("media request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		payload, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("media service error: %s", strings.TrimSpace(string(payload)))
	}

	var success apt.SuccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&success); err != nil {
		return nil, fmt.Errorf("cannot decode media response: %w", err)
	}

	media, err := parseMediaFromMap(success.Data)
	if err != nil {
		return nil, err
	}

	return media, nil
}

// Get retrieves a single media asset by ID.
func (r *APIMediaRepo) Get(ctx context.Context, id uuid.UUID) (*Media, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("media repository not configured")
	}
	resp, err := r.client.Request(ctx, http.MethodGet, fmt.Sprintf("/media/%s", id.String()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}
	if resp == nil || resp.Data == nil {
		return nil, fmt.Errorf("media response payload missing")
	}
	return parseMediaFromMap(resp.Data)
}

// Update modifies mutable attributes of a media asset.
func (r *APIMediaRepo) Update(ctx context.Context, id uuid.UUID, req UpdateMediaRequest) (*Media, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("media repository not configured")
	}

	payload := make(map[string]interface{})
	if req.MimeType != nil {
		payload["mime_type"] = *req.MimeType
	}
	if req.Resolution != nil {
		payload["resolution"] = map[string]int{
			"width":  req.Resolution.Width,
			"height": req.Resolution.Height,
		}
	}
	if req.Filesize != nil {
		payload["filesize"] = *req.Filesize
	}
	if req.Kind != nil {
		payload["kind"] = *req.Kind
	}
	if req.CategoryID != nil {
		payload["category_id"] = req.CategoryID.String()
	}
	if req.Tags != nil {
		tags := make([]string, 0, len(*req.Tags))
		for _, tag := range *req.Tags {
			tags = append(tags, tag.String())
		}
		payload["tags"] = tags
	}
	if req.Metadata != nil {
		meta := make(map[string]any, len(req.Metadata))
		for k, v := range req.Metadata {
			meta[k] = v
		}
		payload["metadata"] = meta
	}
	if req.Enabled != nil {
		payload["enabled"] = *req.Enabled
	}
	if req.StoragePath != nil {
		payload["storage_path"] = *req.StoragePath
	}

	if len(payload) == 0 {
		return r.Get(ctx, id)
	}

	resp, err := r.client.Request(ctx, http.MethodPut, fmt.Sprintf("/media/%s", id.String()), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to update media: %w", err)
	}
	if resp == nil || resp.Data == nil {
		return nil, fmt.Errorf("media update response missing")
	}
	return parseMediaFromMap(resp.Data)
}

// Delete removes a media asset.
func (r *APIMediaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("media repository not configured")
	}
	if _, err := r.client.Request(ctx, http.MethodDelete, fmt.Sprintf("/media/%s", id.String()), nil); err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}
	return nil
}

// Enable toggles a media asset on.
func (r *APIMediaRepo) Enable(ctx context.Context, id uuid.UUID) (*Media, error) {
	return r.toggle(ctx, id, true)
}

// Disable toggles a media asset off.
func (r *APIMediaRepo) Disable(ctx context.Context, id uuid.UUID) (*Media, error) {
	return r.toggle(ctx, id, false)
}

func (r *APIMediaRepo) toggle(ctx context.Context, id uuid.UUID, enabled bool) (*Media, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("media repository not configured")
	}
	endpoint := "disable"
	if enabled {
		endpoint = "enable"
	}
	resp, err := r.client.Request(ctx, http.MethodPost, fmt.Sprintf("/media/%s/%s", id.String(), endpoint), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle media: %w", err)
	}
	if resp == nil || resp.Data == nil {
		return nil, fmt.Errorf("media toggle response missing")
	}
	return parseMediaFromMap(resp.Data)
}

func parseMediaSlice(input interface{}) ([]*Media, error) {
	if input == nil {
		return []*Media{}, nil
	}
	items, ok := input.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid media response format")
	}

	result := make([]*Media, 0, len(items))
	for _, item := range items {
		media, err := parseMediaFromMap(item)
		if err != nil {
			continue
		}
		result = append(result, media)
	}
	return result, nil
}

func parseMediaFromMap(input interface{}) (*Media, error) {
	data, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid media payload")
	}

	idStr := stringField(data, "id")
	if idStr == "" {
		return nil, fmt.Errorf("media id missing")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid media id: %w", err)
	}

	targetID, _ := uuid.Parse(stringField(data, "resource_id"))
	categoryID, _ := uuid.Parse(stringField(data, "category_id"))

	media := &Media{
		ID:                 id,
		TargetType:         stringField(data, "resource_type"),
		TargetID:           targetID,
		StoragePath:        stringField(data, "storage_path"),
		MimeType:           stringField(data, "mime_type"),
		Filesize:           int64(floatField(data, "filesize")),
		Enabled:            boolField(data, "enabled"),
		Kind:               stringField(data, "kind"),
		CategoryID:         categoryID,
		CroppingApplied:    boolField(data, "cropping_applied"),
		CompressionApplied: boolField(data, "compression_applied"),
		Metadata:           mapField(data, "metadata"),
		Variants:           parseMediaVariants(data["variants"]),
		CreatedAt:          timeFieldFromMap(data, "created_at"),
		UpdatedAt:          timeFieldFromMap(data, "updated_at"),
	}

	if resData, ok := data["resolution"].(map[string]interface{}); ok {
		media.Resolution = MediaResolution{
			Width:  int(floatField(resData, "width")),
			Height: int(floatField(resData, "height")),
		}
	}

	if tagsData, ok := data["tags"].([]interface{}); ok {
		media.Tags = make([]uuid.UUID, 0, len(tagsData))
		for _, raw := range tagsData {
			if tagStr, ok := raw.(string); ok {
				if tagID, err := uuid.Parse(tagStr); err == nil {
					media.Tags = append(media.Tags, tagID)
				}
			}
		}
	}

	return media, nil
}

func parseMediaVariants(raw interface{}) map[string]MediaVariant {
	variantsData, ok := raw.(map[string]interface{})
	if !ok || len(variantsData) == 0 {
		return nil
	}

	result := make(map[string]MediaVariant, len(variantsData))
	for name, payload := range variantsData {
		vMap, ok := payload.(map[string]interface{})
		if !ok {
			continue
		}
		result[name] = MediaVariant{
			Path:     stringField(vMap, "path"),
			MimeType: stringField(vMap, "mime_type"),
			Width:    int(floatField(vMap, "width")),
			Height:   int(floatField(vMap, "height")),
			Filesize: int64(floatField(vMap, "filesize")),
		}
	}

	return result
}

func mapField(data map[string]interface{}, key string) map[string]any {
	raw, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]any, len(raw))
	for k, v := range raw {
		result[k] = v
	}
	return result
}

func timeFieldFromMap(data map[string]interface{}, key string) time.Time {
	if v, ok := data[key].(string); ok && v != "" {
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
