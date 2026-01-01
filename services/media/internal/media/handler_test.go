package media

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/appetiteclub/apt"
)

func setupHandler(t *testing.T) (*Handler, *stubDictionary, *stubStorage) {
	t.Helper()
	dict := &stubDictionary{}
	store := &stubStorage{path: "stored/http"}
	service := NewService(NewInMemoryRepository(), store, dict, ServiceOptions{})
	deps := apt.DefaultDeps()
	return NewHandler(service, deps), dict, store
}

func TestHandlerFlow(t *testing.T) {
	handler, dict, store := setupHandler(t)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	resourceID := uuid.New()
	categoryID := uuid.New()
	tagID := uuid.New()

	payload := fmt.Sprintf(`{
        "resource_type": "property",
        "resource_id": "%s",
        "mime_type": "image/jpeg",
        "resolution": {"width": 800, "height": 600},
        "filesize": 1234,
        "kind": "real",
        "category_id": "%s",
        "tags": ["%s"],
        "storage_path": "existing/path",
        "enabled": true
    }`, resourceID, categoryID, tagID)

	req := httptest.NewRequest(http.MethodPost, "/media", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		Data Media `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("cannot decode create response: %v", err)
	}

	if created.Data.TargetType != "property" {
		t.Fatalf("unexpected target type %s", created.Data.TargetType)
	}
	if len(dict.categories) != 1 || dict.categories[0] != categoryID {
		t.Fatalf("expected category validation")
	}
	if len(store.saves) != 0 {
		t.Fatalf("HTTP create should not trigger storage save when storage_path provided")
	}

	// List
	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media?resource_type=property&resource_id=%s", resourceID), nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listRec.Code)
	}
	var listResp struct {
		Data []Media `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("cannot decode list response: %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("expected one media, got %d", len(listResp.Data))
	}

	// Update kind
	newKindPayload := fmt.Sprintf(`{"kind": "ai_generated", "enabled": false}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/media/"+created.Data.ID.String(), bytes.NewBufferString(newKindPayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d", updateRec.Code)
	}
	var updateResp struct {
		Data Media `json:"data"`
	}
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("cannot decode update response: %v", err)
	}
	if updateResp.Data.Kind != KindAIGenerated {
		t.Fatalf("expected kind to change, got %s", updateResp.Data.Kind)
	}
	if updateResp.Data.Enabled {
		t.Fatalf("expected media to be disabled")
	}

	// Disable explicitly
	disableReq := httptest.NewRequest(http.MethodPost, "/media/"+created.Data.ID.String()+"/disable", nil)
	disableRec := httptest.NewRecorder()
	router.ServeHTTP(disableRec, disableReq)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on disable, got %d", disableRec.Code)
	}

	// Delete
	delReq := httptest.NewRequest(http.MethodDelete, "/media/"+created.Data.ID.String(), nil)
	delRec := httptest.NewRecorder()
	router.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete, got %d", delRec.Code)
	}
	if len(store.deletes) != 1 || store.deletes[0] != "existing/path" {
		t.Fatalf("expected storage delete of existing/path")
	}

	// Validation error
	badReq := httptest.NewRequest(http.MethodPost, "/media", bytes.NewBufferString(`{"resource_type":"","resource_id":"bad"}`))
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", badRec.Code)
	}
}

func TestHandlerInvalidID(t *testing.T) {
	handler, _, _ := setupHandler(t)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/media/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d", rec.Code)
	}
}

func TestHandlerUpdateInvalidCategory(t *testing.T) {
	dict := &stubDictionary{failCat: true}
	store := &stubStorage{}
	svc := NewService(NewInMemoryRepository(), store, dict, ServiceOptions{})
	deps := apt.DefaultDeps()
	handler := NewHandler(svc, deps)

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	resourceID := uuid.New()
	categoryID := uuid.New()

	// First create without failing category
	dict.failCat = false
	payload := fmt.Sprintf(`{"resource_type":"property","resource_id":"%s","mime_type":"image/jpeg","resolution":{"width":10,"height":10},"filesize":10,"kind":"real","category_id":"%s"}`, resourceID, categoryID)
	req := httptest.NewRequest(http.MethodPost, "/media", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
	}
	var created struct{ Data Media }
	json.Unmarshal(rec.Body.Bytes(), &created)

	// Now update with failing category
	dict.failCat = true
	newCat := uuid.New()
	updatePayload := fmt.Sprintf(`{"category_id":"%s"}`, newCat)
	upReq := httptest.NewRequest(http.MethodPut, "/media/"+created.Data.ID.String(), bytes.NewBufferString(updatePayload))
	upReq.Header.Set("Content-Type", "application/json")
	upRec := httptest.NewRecorder()
	router.ServeHTTP(upRec, upReq)
	if upRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when dictionary validation fails, got %d", upRec.Code)
	}
}

func TestHandlerListMissingParams(t *testing.T) {
	handler, _, _ := setupHandler(t)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/media", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even with empty filters, got %d", rec.Code)
	}
	var resp struct {
		Data []Media `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("cannot decode list response: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Fatalf("expected empty list by default")
	}
}

func TestHandlerDeleteNotFound(t *testing.T) {
	handler, _, _ := setupHandler(t)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/media/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when deleting missing media, got %d", rec.Code)
	}
}

func TestHandlerToggleNotFound(t *testing.T) {
	handler, _, _ := setupHandler(t)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/media/"+uuid.New().String()+"/disable", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 toggle missing media, got %d", rec.Code)
	}
}
