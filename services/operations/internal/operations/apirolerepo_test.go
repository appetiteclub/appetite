package operations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestNewAPIRoleRepoNilConfig(t *testing.T) {
	_, err := NewAPIRoleRepo(nil, nil)
	if err == nil {
		t.Error("NewAPIRoleRepo() with nil config should return error")
	}
}

func TestAPIRoleRepoGet(t *testing.T) {
	roleID := uuid.New()

	role := &Role{
		ID:          roleID,
		Name:        "admin",
		Description: "Administrator",
		Permissions: []string{"read", "write"},
		Status:      "active",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": role,
		})
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	result, err := repo.Get(context.Background(), roleID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result == nil {
		t.Fatal("Get() returned nil")
	}
	if result.ID != roleID {
		t.Errorf("ID = %v, want %v", result.ID, roleID)
	}
	if result.Name != "admin" {
		t.Errorf("Name = %q, want %q", result.Name, "admin")
	}
}

func TestAPIRoleRepoGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.Get(context.Background(), uuid.New())
	if err == nil {
		t.Error("Get() with 404 should return error")
	}
}

func TestAPIRoleRepoList(t *testing.T) {
	roles := []*Role{
		{ID: uuid.New(), Name: "admin", Status: "active"},
		{ID: uuid.New(), Name: "user", Status: "active"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": roles,
		})
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	result, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("List() returned %d roles, want 2", len(result))
	}
}

func TestAPIRoleRepoListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.List(context.Background())
	if err == nil {
		t.Error("List() with 500 should return error")
	}
}

func TestAPIRoleRepoListByStatus(t *testing.T) {
	roles := []*Role{
		{ID: uuid.New(), Name: "admin", Status: "active"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %s, want GET", r.Method)
		}

		// Check query parameter
		status := r.URL.Query().Get("status")
		if status != "active" {
			t.Errorf("status query param = %q, want %q", status, "active")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": roles,
		})
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	result, err := repo.ListByStatus(context.Background(), "active")
	if err != nil {
		t.Fatalf("ListByStatus() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("ListByStatus() returned %d roles, want 1", len(result))
	}
}

func TestAPIRoleRepoListByStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.ListByStatus(context.Background(), "active")
	if err == nil {
		t.Error("ListByStatus() with 500 should return error")
	}
}

func TestAPIRoleRepoGetInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.Get(context.Background(), uuid.New())
	if err == nil {
		t.Error("Get() with invalid JSON should return error")
	}
}

func TestAPIRoleRepoListInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.List(context.Background())
	if err == nil {
		t.Error("List() with invalid JSON should return error")
	}
}

func TestAPIRoleRepoListByStatusInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	repo := &APIRoleRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.ListByStatus(context.Background(), "active")
	if err == nil {
		t.Error("ListByStatus() with invalid JSON should return error")
	}
}
