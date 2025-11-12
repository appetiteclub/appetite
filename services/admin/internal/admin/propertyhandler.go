package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/aquamarinepk/aqm"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxMediaUploadSize = 32 << 20 // 32MB

// ListProperties shows all properties
func (h *Handler) ListProperties(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.ListProperties")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	properties, err := h.service.ListProperties(ctx)
	if err != nil {
		log.Error("error listing properties", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	priceTypes, err := h.dictRepo.ListPriceTypes(ctx)
	if err != nil {
		log.Error("error fetching price types", "error", err)
	}

	tmpl, err := h.tmplMgr.Get("list-properties.html")
	if err != nil {
		log.Error("error getting template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":           "Properties",
		"Properties":      properties,
		"ActiveNav":       "properties",
		"Template":        "list-properties-content",
		"PriceTypeLabels": map[string]string{},
	}

	if priceTypes != nil {
		data["PriceTypeLabels"] = priceLabelsByKey(priceTypes)
	}

	if err := tmpl.ExecuteTemplate(w, "list-properties.html", data); err != nil {
		log.Error("error executing template", "error", err)
	}
}

// NewProperty shows the form to create a new property
func (h *Handler) NewProperty(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.NewProperty")
	defer finish()
	log := h.log(r)

	ctx := r.Context()

	// Fetch fake options
	categories, err := h.dictRepo.ListCategories(ctx)
	if err != nil {
		log.Error("error fetching categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	types, err := h.dictRepo.ListTypesByCategory(ctx, uuid.Nil)
	if err != nil {
		log.Error("error fetching types", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	subtypes, err := h.dictRepo.ListSubtypesByType(ctx, uuid.Nil)
	if err != nil {
		log.Error("error fetching subtypes", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	statuses, err := h.dictRepo.ListStatuses(ctx)
	if err != nil {
		log.Error("error fetching statuses", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	priceTypes, err := h.dictRepo.ListPriceTypes(ctx)
	if err != nil {
		log.Error("error fetching price types", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	conditions, err := h.dictRepo.ListConditions(ctx)
	if err != nil {
		log.Error("error fetching conditions", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := h.tmplMgr.Get("new-property.html")
	if err != nil {
		log.Error("error getting template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":           "New Property",
		"ActiveNav":       "properties",
		"Template":        "new-property",
		"Categories":      DictionaryOptionsToMap(categories),
		"Types":           DictionaryOptionsToMap(types),
		"Subtypes":        DictionaryOptionsToMap(subtypes),
		"Statuses":        DictionaryOptionsToMap(statuses),
		"PriceTypes":      DictionaryOptionsToMap(priceTypes),
		"Conditions":      DictionaryOptionsToMap(conditions),
		"Location":        newLocationFormModel(),
		"PriceValues":     map[string]*Price{},
		"PriceTypeLabels": priceLabelsByKey(priceTypes),
	}

	if err := tmpl.ExecuteTemplate(w, "new-property.html", data); err != nil {
		log.Error("error executing template", "error", err)
	}
}

// CreateProperty handles the creation of a new property
func (h *Handler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CreateProperty")
	defer finish()
	log := h.log(r)

	ctx := r.Context()

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Error("error parsing form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Parse classification IDs
	categoryID, _ := uuid.Parse(r.FormValue("category_id"))
	typeID, _ := uuid.Parse(r.FormValue("type_id"))
	subtypeIDStr := r.FormValue("subtype_id")
	var subtypeID uuid.UUID
	if subtypeIDStr != "" {
		subtypeID, _ = uuid.Parse(subtypeIDStr)
	}

	// Parse numeric fields
	totalArea, _ := strconv.ParseFloat(r.FormValue("total_area"), 64)
	bedrooms, _ := strconv.Atoi(r.FormValue("bedrooms"))
	bathrooms, _ := strconv.Atoi(r.FormValue("bathrooms"))
	parking, _ := strconv.Atoi(r.FormValue("parking"))

	prices := extractPricesFromForm(r)

	location := Location{
		Address: Address{
			Street:     strings.TrimSpace(r.FormValue("street")),
			Number:     strings.TrimSpace(r.FormValue("number")),
			Unit:       strings.TrimSpace(r.FormValue("unit")),
			City:       strings.TrimSpace(r.FormValue("city")),
			State:      strings.TrimSpace(r.FormValue("state")),
			PostalCode: strings.TrimSpace(r.FormValue("postal_code")),
			Country:    strings.TrimSpace(r.FormValue("country")),
		},
		Coordinates: Coordinates{
			Latitude:  parseCoordinateValue(r.FormValue("location_latitude")),
			Longitude: parseCoordinateValue(r.FormValue("location_longitude")),
		},
		Region:      strings.TrimSpace(r.FormValue("region")),
		Provider:    strings.TrimSpace(r.FormValue("location_provider")),
		ProviderRef: strings.TrimSpace(r.FormValue("location_provider_ref")),
		ProviderURL: strings.TrimSpace(r.FormValue("location_provider_url")),
		Raw:         parseLocationRaw(r.FormValue("location_raw")),
		DisplayName: strings.TrimSpace(r.FormValue("location_display_name")),
	}
	if len(location.Raw) == 0 {
		location.Raw = nil
	}

	req := &CreatePropertyRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Classification: Classification{
			CategoryID: categoryID,
			TypeID:     typeID,
			SubtypeID:  subtypeID,
		},
		Location: location,
		Features: Features{
			TotalArea: totalArea,
			Bedrooms:  bedrooms,
			Bathrooms: bathrooms,
			Parking:   parking,
		},
		Prices:        prices,
		Status:        strings.TrimSpace(r.FormValue("status")),
		OwnerID:       strings.TrimSpace(r.FormValue("owner_id")),
		SchemaVersion: CurrentPropertySchemaVersion,
	}

	property, err := h.service.CreateProperty(ctx, req)
	if err != nil {
		log.Error("error creating property", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Info("property created successfully", "id", property.ID)
	http.Redirect(w, r, "/list-properties", http.StatusSeeOther)
}

// SuggestLocations returns autocomplete results for property addresses.
func (h *Handler) SuggestLocations(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.SuggestLocations")
	defer finish()
	log := h.log(r)

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		aqm.RespondError(w, http.StatusBadRequest, "query parameter q is required")
		return
	}

	suggestions, err := h.service.SuggestLocations(r.Context(), query)
	if err != nil {
		if errors.Is(err, ErrLocationProviderUnavailable) {
			aqm.RespondSuccess(w, []LocationSuggestion{})
			return
		}
		log.Error("error fetching location suggestions", "error", err)
		aqm.RespondError(w, http.StatusBadGateway, "Could not fetch suggestions")
		return
	}

	aqm.RespondSuccess(w, suggestions)
}

// HTMXNormalizeLocation resolves and normalizes the selected location into form fields.
func (h *Handler) HTMXNormalizeLocation(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.HTMXNormalizeLocation")
	defer finish()
	log := h.log(r)

	if err := r.ParseForm(); err != nil {
		log.Error("error parsing normalize location form", "error", err)
		model := locationFormModelFromRequest(r)
		model.Error = "Could not parse location data"
		h.renderLocationFragment(w, model)
		return
	}

	providerRef := strings.TrimSpace(r.FormValue("provider_ref"))
	selectedText := strings.TrimSpace(r.FormValue("selected_text"))
	if providerRef == "" {
		model := locationFormModelFromRequest(r)
		model.Error = "Missing location identifier"
		h.renderLocationFragment(w, model)
		return
	}

	normalized, err := h.service.NormalizeLocation(r.Context(), NormalizeLocationRequest{
		ProviderRef:  providerRef,
		SelectedText: selectedText,
	})
	if err != nil {
		model := locationFormModelFromRequest(r)
		if errors.Is(err, ErrLocationProviderUnavailable) {
			model.Error = "Location provider not configured"
		} else {
			log.Error("error normalizing location", "error", err, "provider_ref", providerRef)
			model.Error = "Could not normalize location"
		}
		h.renderLocationFragment(w, model)
		return
	}

	model := locationFormModelFromNormalized(normalized)
	emitLocationUpdateTrigger(w, model, log)
	h.renderLocationFragment(w, model)
}

func (h *Handler) renderLocationFragment(w http.ResponseWriter, model LocationFormModel) {
	tmpl, err := h.tmplMgr.Get("location-fields.html")
	if err != nil {
		h.log().Error("location fields template not found", "error", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Location": model,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "location-fields.html", data); err != nil {
		h.log().Error("error rendering location fields", "error", err)
	}
}

func parseCoordinateValue(value string) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseLocationRaw(raw string) map[string]any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return nil
	}
	return result
}

func emitLocationUpdateTrigger(w http.ResponseWriter, model LocationFormModel, log aqm.Logger) {
	payload := map[string]string{
		"search_value":  model.SearchValue,
		"selected_text": model.SelectedText,
		"street":        model.Street,
		"number":        model.Number,
		"unit":          model.Unit,
		"city":          model.City,
		"state":         model.State,
		"postal_code":   model.PostalCode,
		"country":       model.Country,
		"latitude":      model.Latitude,
		"longitude":     model.Longitude,
		"provider":      model.Provider,
		"provider_ref":  model.ProviderRef,
		"provider_url":  model.ProviderURL,
		"raw_json":      model.RawJSON,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if log != nil {
		log.Info("location fields normalized",
			"search", model.SearchValue,
			"selected", model.SelectedText,
			"street", model.Street,
			"number", model.Number,
			"city", model.City,
			"state", model.State,
			"postal_code", model.PostalCode,
			"country", model.Country,
			"latitude", model.Latitude,
			"longitude", model.Longitude,
		)
	}
	w.Header().Set("HX-Trigger-After-Swap", fmt.Sprintf("{\"locationUpdated\":%s}", data))
}

// ShowProperty displays a single property
func (h *Handler) ShowProperty(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.ShowProperty")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid property id", "id", idStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	property, err := h.service.GetProperty(ctx, id)
	if err != nil {
		log.Error("error getting property", "error", err, "id", id)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	mediaAssets, err := h.service.ListPropertyMedia(ctx, id)
	if err != nil {
		log.Error("error fetching media for property", "error", err, "id", id)
		mediaAssets = []*Media{}
	}

	priceTypes, err := h.dictRepo.ListPriceTypes(ctx)
	if err != nil {
		log.Error("error fetching price types", "error", err)
	}

	mediaCategories, mediaKinds, mediaTags := h.mediaLabelMaps(ctx)

	tmpl, err := h.tmplMgr.Get("show-property.html")
	if err != nil {
		log.Error("error getting template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":               fmt.Sprintf("Property: %s", property.Name),
		"Property":            property,
		"ActiveNav":           "properties",
		"Template":            "show-property",
		"PriceTypeLabels":     map[string]string{},
		"PropertyMedia":       mediaAssets,
		"MediaCategoryLabels": mediaCategories,
		"MediaKindLabels":     mediaKinds,
		"MediaTagLabels":      mediaTags,
	}

	if priceTypes != nil {
		data["PriceTypeLabels"] = priceLabelsByKey(priceTypes)
	}
	data["PriceValues"] = priceValuesByType(property.Prices)

	if err := tmpl.ExecuteTemplate(w, "show-property.html", data); err != nil {
		log.Error("error executing template", "error", err)
	}
}

// EditProperty shows the form to edit a property
func (h *Handler) EditProperty(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.EditProperty")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid property id", "id", idStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	property, err := h.service.GetProperty(ctx, id)
	if err != nil {
		log.Error("error getting property", "error", err, "id", id)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Fetch fake options
	categories, err := h.dictRepo.ListCategories(ctx)
	if err != nil {
		log.Error("error fetching categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	types, err := h.dictRepo.ListTypesByCategory(ctx, uuid.Nil)
	if err != nil {
		log.Error("error fetching types", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	subtypes, err := h.dictRepo.ListSubtypesByType(ctx, uuid.Nil)
	if err != nil {
		log.Error("error fetching subtypes", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	statuses, err := h.dictRepo.ListStatuses(ctx)
	if err != nil {
		log.Error("error fetching statuses", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	priceTypes, err := h.dictRepo.ListPriceTypes(ctx)
	if err != nil {
		log.Error("error fetching price types", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	conditions, err := h.dictRepo.ListConditions(ctx)
	if err != nil {
		log.Error("error fetching conditions", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	mediaAssets, err := h.service.ListPropertyMedia(ctx, id)
	if err != nil {
		log.Error("error fetching media for property", "error", err, "id", id)
		mediaAssets = []*Media{}
	}

	mediaCategories, mediaKinds, mediaTags := h.mediaLabelMaps(ctx)
	mediaOptions := h.mediaFormOptions(ctx)

	tmpl, err := h.tmplMgr.Get("edit-property.html")
	if err != nil {
		log.Error("error getting template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":               fmt.Sprintf("Edit: %s", property.Name),
		"Property":            property,
		"ActiveNav":           "properties",
		"Template":            "edit-property",
		"Categories":          DictionaryOptionsToMap(categories),
		"Types":               DictionaryOptionsToMap(types),
		"Subtypes":            DictionaryOptionsToMap(subtypes),
		"Statuses":            DictionaryOptionsToMap(statuses),
		"PriceTypes":          DictionaryOptionsToMap(priceTypes),
		"Conditions":          DictionaryOptionsToMap(conditions),
		"Location":            locationFormModelFromProperty(property),
		"PriceValues":         priceValuesByType(property.Prices),
		"PriceTypeLabels":     priceLabelsByKey(priceTypes),
		"PropertyMedia":       mediaAssets,
		"MediaCategoryLabels": mediaCategories,
		"MediaKindLabels":     mediaKinds,
		"MediaTagLabels":      mediaTags,
		"MediaCategories":     mediaOptions.Categories,
		"MediaKinds":          mediaOptions.Kinds,
		"MediaTagGroups":      mediaOptions.TagGroups,
	}

	if err := tmpl.ExecuteTemplate(w, "edit-property.html", data); err != nil {
		log.Error("error executing template", "error", err)
	}
}

// UpdateProperty handles updating an existing property
func (h *Handler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.UpdateProperty")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid property id", "id", idStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Error("error parsing form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Parse classification IDs
	categoryID, _ := uuid.Parse(r.FormValue("category_id"))
	typeID, _ := uuid.Parse(r.FormValue("type_id"))
	subtypeIDStr := r.FormValue("subtype_id")
	var subtypeID uuid.UUID
	if subtypeIDStr != "" {
		subtypeID, _ = uuid.Parse(subtypeIDStr)
	}

	// Parse numeric fields
	totalArea, _ := strconv.ParseFloat(r.FormValue("total_area"), 64)
	bedrooms, _ := strconv.Atoi(r.FormValue("bedrooms"))
	bathrooms, _ := strconv.Atoi(r.FormValue("bathrooms"))
	parking, _ := strconv.Atoi(r.FormValue("parking"))

	prices := extractPricesFromForm(r)

	location := Location{
		Address: Address{
			Street:     strings.TrimSpace(r.FormValue("street")),
			Number:     strings.TrimSpace(r.FormValue("number")),
			Unit:       strings.TrimSpace(r.FormValue("unit")),
			City:       strings.TrimSpace(r.FormValue("city")),
			State:      strings.TrimSpace(r.FormValue("state")),
			PostalCode: strings.TrimSpace(r.FormValue("postal_code")),
			Country:    strings.TrimSpace(r.FormValue("country")),
		},
		Coordinates: Coordinates{
			Latitude:  parseCoordinateValue(r.FormValue("location_latitude")),
			Longitude: parseCoordinateValue(r.FormValue("location_longitude")),
		},
		Region:      strings.TrimSpace(r.FormValue("region")),
		Provider:    strings.TrimSpace(r.FormValue("location_provider")),
		ProviderRef: strings.TrimSpace(r.FormValue("location_provider_ref")),
		ProviderURL: strings.TrimSpace(r.FormValue("location_provider_url")),
		Raw:         parseLocationRaw(r.FormValue("location_raw")),
		DisplayName: strings.TrimSpace(r.FormValue("location_display_name")),
	}
	if len(location.Raw) == 0 {
		location.Raw = nil
	}

	req := &UpdatePropertyRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Classification: Classification{
			CategoryID: categoryID,
			TypeID:     typeID,
			SubtypeID:  subtypeID,
		},
		Location: location,
		Features: Features{
			TotalArea: totalArea,
			Bedrooms:  bedrooms,
			Bathrooms: bathrooms,
			Parking:   parking,
		},
		Prices:        prices,
		Status:        strings.TrimSpace(r.FormValue("status")),
		OwnerID:       strings.TrimSpace(r.FormValue("owner_id")),
		SchemaVersion: CurrentPropertySchemaVersion,
	}

	_, err = h.service.UpdateProperty(ctx, id, req)
	if err != nil {
		log.Error("error updating property", "error", err, "id", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Info("property updated successfully", "id", id)
	http.Redirect(w, r, fmt.Sprintf("/show-property/%s", id), http.StatusSeeOther)
}

// CreatePropertyMedia handles uploading a new media asset for a property.
func (h *Handler) CreatePropertyMedia(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CreatePropertyMedia")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	propertyID, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid property id", "id", idStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(maxMediaUploadSize); err != nil {
		log.Error("error parsing media form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Error("media file missing", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	buffer := &bytes.Buffer{}
	size, err := io.Copy(buffer, file)
	if err != nil {
		log.Error("error reading media file", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if size == 0 {
		http.Error(w, "Uploaded file is empty", http.StatusBadRequest)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(buffer.Bytes())
	}

	width := parseIntDefault(r.FormValue("resolution_width"), 0)
	height := parseIntDefault(r.FormValue("resolution_height"), 0)
	if width == 0 || height == 0 {
		cfg, format, err := image.DecodeConfig(bytes.NewReader(buffer.Bytes()))
		if err == nil {
			width = cfg.Width
			height = cfg.Height
			if mimeType == "" {
				mimeType = "image/" + format
			}
		} else {
			log.Error("cannot determine image dimensions", "error", err)
			http.Error(w, "Unable to determine image dimensions", http.StatusBadRequest)
			return
		}
	}

	categoryID, err := uuid.Parse(strings.TrimSpace(r.FormValue("media_category_id")))
	if err != nil {
		http.Error(w, "Invalid media category", http.StatusBadRequest)
		return
	}

	kind := strings.TrimSpace(r.FormValue("media_kind"))
	if kind == "" {
		kind = "real"
	}

	enabled := true
	if r.FormValue("media_enabled") == "" {
		enabled = false
	}

	tagIDs := gatherMediaTagIDs(r.MultipartForm, DefaultMediaTagSetNames()...)

	metadata := map[string]string{
		"original_name": header.Filename,
	}

	request := CreateMediaRequest{
		ResourceType: "property",
		ResourceID:   propertyID,
		FileName:     header.Filename,
		MimeType:     mimeType,
		Data:         buffer.Bytes(),
		Filesize:     size,
		CategoryID:   categoryID,
		Kind:         kind,
		Tags:         tagIDs,
		Enabled:      enabled,
		Resolution:   MediaResolution{Width: width, Height: height},
		Metadata:     metadata,
	}

	if _, err := h.service.CreatePropertyMedia(ctx, propertyID, request); err != nil {
		log.Error("error creating property media", "error", err, "property_id", propertyID)
		http.Error(w, "Could not upload media", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/edit-property/%s", propertyID.String()), http.StatusSeeOther)
}

// DeleteProperty handles deleting a property
func (h *Handler) DeleteProperty(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.DeleteProperty")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("invalid property id", "id", idStr)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteProperty(ctx, id); err != nil {
		log.Error("error deleting property", "error", err, "id", id)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Info("property deleted successfully", "id", id)
	http.Redirect(w, r, "/list-properties", http.StatusSeeOther)
}

// HTMXTypesByCategory returns HTML options for types filtered by category
func (h *Handler) HTMXTypesByCategory(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.HTMXTypesByCategory")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	categoryIDStr := r.URL.Query().Get("category_id")

	var categoryID uuid.UUID
	if categoryIDStr != "" {
		var err error
		categoryID, err = uuid.Parse(categoryIDStr)
		if err != nil {
			log.Error("invalid category id", "category_id", categoryIDStr)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<option value="">-- Invalid Category --</option>`))
			return
		}
	}

	types, err := h.dictRepo.ListTypesByCategory(ctx, categoryID)
	if err != nil {
		log.Error("error fetching types", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<option value="">-- Error loading types --</option>`))
		return
	}

	// Write HTML options
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<option value="">-- Select Type --</option>`))
	for _, t := range types {
		w.Write([]byte(fmt.Sprintf(`<option value="%s">%s</option>`, t.ID.String(), t.Name)))
	}
}

// HTMXSubtypesByType returns HTML options for subtypes filtered by type
func (h *Handler) HTMXSubtypesByType(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.HTMXSubtypesByType")
	defer finish()
	log := h.log(r)

	ctx := r.Context()
	typeIDStr := r.URL.Query().Get("type_id")

	var typeID uuid.UUID
	if typeIDStr != "" {
		var err error
		typeID, err = uuid.Parse(typeIDStr)
		if err != nil {
			log.Error("invalid type id", "type_id", typeIDStr)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<option value="">-- Invalid Type --</option>`))
			return
		}
	}

	subtypes, err := h.dictRepo.ListSubtypesByType(ctx, typeID)
	if err != nil {
		log.Error("error fetching subtypes", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<option value="">-- Error loading subtypes --</option>`))
		return
	}

	// Write HTML options
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<option value="">-- Select Subtype (Optional) --</option>`))
	for _, s := range subtypes {
		w.Write([]byte(fmt.Sprintf(`<option value="%s">%s</option>`, s.ID.String(), s.Name)))
	}
}

// extractPricesFromForm collects price rows submitted via the property form.
func extractPricesFromForm(r *http.Request) []Price {
	type rawPrice struct {
		amount     string
		currency   string
		negotiable bool
	}

	pricesByType := make(map[string]rawPrice)

	for key, values := range r.PostForm {
		if !strings.HasPrefix(key, "price_") || !strings.HasSuffix(key, "_amount") {
			continue
		}

		typeKey := strings.TrimSuffix(strings.TrimPrefix(key, "price_"), "_amount")
		if typeKey == "" {
			continue
		}

		amount := ""
		if len(values) > 0 {
			amount = strings.TrimSpace(values[0])
		}

		currency := strings.TrimSpace(r.PostFormValue(fmt.Sprintf("price_%s_currency", typeKey)))
		negotiable := r.PostFormValue(fmt.Sprintf("price_%s_negotiable", typeKey)) == "on"

		pricesByType[typeKey] = rawPrice{
			amount:     amount,
			currency:   currency,
			negotiable: negotiable,
		}
	}

	// Legacy single-price fallback for older forms
	if len(pricesByType) == 0 {
		legacyAmount := strings.TrimSpace(r.FormValue("price_amount"))
		if legacyAmount != "" {
			pricesByType[strings.TrimSpace(r.FormValue("price_type"))] = rawPrice{
				amount:     legacyAmount,
				currency:   strings.TrimSpace(r.FormValue("currency")),
				negotiable: r.FormValue("price_negotiable") == "on",
			}
		}
	}

	if len(pricesByType) == 0 {
		return nil
	}

	// Sort keys so output order is deterministic
	typeKeys := make([]string, 0, len(pricesByType))
	for key := range pricesByType {
		if strings.TrimSpace(key) == "" {
			continue
		}
		typeKeys = append(typeKeys, key)
	}
	sort.Strings(typeKeys)

	prices := make([]Price, 0, len(typeKeys))
	for _, key := range typeKeys {
		entry := pricesByType[key]
		if entry.amount == "" {
			continue
		}

		amount, _ := strconv.ParseFloat(entry.amount, 64)
		prices = append(prices, Price{
			Amount:     amount,
			Currency:   entry.currency,
			Type:       key,
			Negotiable: entry.negotiable,
		})
	}

	return prices
}

func parseIntDefault(value string, def int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	if num, err := strconv.Atoi(value); err == nil {
		return num
	}
	return def
}

func gatherMediaTagIDs(form *multipart.Form, fields ...string) []uuid.UUID {
	if form == nil {
		return nil
	}
	seen := make(map[uuid.UUID]struct{})
	var ids []uuid.UUID
	for _, field := range fields {
		values := form.Value[field]
		for _, raw := range values {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			tagID, err := uuid.Parse(raw)
			if err != nil {
				continue
			}
			if _, ok := seen[tagID]; ok {
				continue
			}
			seen[tagID] = struct{}{}
			ids = append(ids, tagID)
		}
	}
	return ids
}

func gatherMediaTagIDsFromValues(values map[string][]string, fields ...string) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{})
	var ids []uuid.UUID
	for _, field := range fields {
		entries := values[field]
		for _, raw := range entries {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			tagID, err := uuid.Parse(raw)
			if err != nil {
				continue
			}
			if _, exists := seen[tagID]; exists {
				continue
			}
			seen[tagID] = struct{}{}
			ids = append(ids, tagID)
		}
	}
	return ids
}

func priceValuesByType(prices []Price) map[string]*Price {
	if len(prices) == 0 {
		return map[string]*Price{}
	}

	result := make(map[string]*Price, len(prices))
	for _, price := range prices {
		if price.Type == "" {
			continue
		}
		p := price
		result[price.Type] = &p
	}
	return result
}

func priceLabelsByKey(options []DictionaryOption) map[string]string {
	labels := make(map[string]string, len(options))
	for _, opt := range options {
		key := strings.TrimSpace(opt.Key)
		if key == "" {
			continue
		}
		labels[key] = opt.Name
	}
	return labels
}

func (h *Handler) mediaLabelMaps(ctx context.Context) (map[string]string, map[string]string, map[string]string) {
	categories := make(map[string]string)
	kinds := make(map[string]string)
	tags := make(map[string]string)

	if h.dictRepo == nil {
		return categories, kinds, tags
	}

	const locale = "en"

	if opts, err := h.dictRepo.GetOptionsBySetName(ctx, "media_location", locale, nil); err == nil {
		for _, opt := range opts {
			categories[opt.ID.String()] = opt.Name
		}
	} else {
		h.log().Error("error fetching media locations", "error", err)
	}

	if opts, err := h.dictRepo.GetOptionsBySetName(ctx, "media_kind", locale, nil); err == nil {
		for _, opt := range opts {
			key := strings.TrimSpace(opt.Key)
			if key == "" {
				key = opt.ID.String()
			}
			kinds[key] = opt.Name
		}
	} else {
		h.log().Error("error fetching media kinds", "error", err)
	}

	for _, setInfo := range DefaultMediaTagSets() {
		opts, err := h.dictRepo.GetOptionsBySetName(ctx, setInfo.Name, locale, nil)
		if err != nil {
			h.log().Error("error fetching media options", "set", setInfo.Name, "error", err)
			continue
		}
		for _, opt := range opts {
			tags[opt.ID.String()] = opt.Name
		}
	}

	return categories, kinds, tags
}

type mediaFormOptions struct {
	Categories []DictionaryOption
	Kinds      []DictionaryOption
	TagGroups  []mediaTagGroup
}

type mediaTagGroup struct {
	ID      string
	Label   string
	Options []DictionaryOption
}

func (h *Handler) mediaFormOptions(ctx context.Context) mediaFormOptions {
	options := mediaFormOptions{}
	if h.dictRepo == nil {
		return options
	}

	const locale = "en"

	if cats, err := h.dictRepo.GetOptionsBySetName(ctx, "media_location", locale, nil); err == nil {
		options.Categories = cats
	} else {
		h.log().Error("error fetching media locations", "error", err)
	}

	if kinds, err := h.dictRepo.GetOptionsBySetName(ctx, "media_kind", locale, nil); err == nil {
		options.Kinds = kinds
	} else {
		h.log().Error("error fetching media kinds", "error", err)
	}

	for _, tagSet := range DefaultMediaTagSets() {
		opts, err := h.dictRepo.GetOptionsBySetName(ctx, tagSet.Name, locale, nil)
		if err != nil {
			h.log().Error("error fetching media tag options", "set", tagSet.Name, "error", err)
			continue
		}
		if len(opts) == 0 {
			continue
		}
		options.TagGroups = append(options.TagGroups, mediaTagGroup{
			ID:      tagSet.Name,
			Label:   tagSet.Label,
			Options: opts,
		})
	}

	return options
}
