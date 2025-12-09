package operations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestNewAPIGrantRepoNilConfig(t *testing.T) {
	_, err := NewAPIGrantRepo(nil, nil)
	if err == nil {
		t.Error("NewAPIGrantRepo() with nil config should return error")
	}
}

func TestAPIGrantRepoGet(t *testing.T) {
	grantID := uuid.New()
	userID := uuid.New()

	grant := &Grant{
		ID:        grantID,
		UserID:    userID,
		GrantType: "role",
		Value:     "admin",
		Status:    "active",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": grant,
		})
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	result, err := repo.Get(context.Background(), grantID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result == nil {
		t.Fatal("Get() returned nil")
	}
	if result.ID != grantID {
		t.Errorf("ID = %v, want %v", result.ID, grantID)
	}
}

func TestAPIGrantRepoGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.Get(context.Background(), uuid.New())
	if err == nil {
		t.Error("Get() with 404 should return error")
	}
}

func TestAPIGrantRepoList(t *testing.T) {
	grants := []*Grant{
		{ID: uuid.New(), GrantType: "role", Value: "admin", Status: "active"},
		{ID: uuid.New(), GrantType: "role", Value: "user", Status: "active"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": grants,
		})
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	result, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("List() returned %d grants, want 2", len(result))
	}
}

func TestAPIGrantRepoListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.List(context.Background())
	if err == nil {
		t.Error("List() with 500 should return error")
	}
}

func TestAPIGrantRepoListByUser(t *testing.T) {
	userID := uuid.New()
	grants := []*Grant{
		{ID: uuid.New(), UserID: userID, GrantType: "role", Value: "admin", Status: "active"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": grants,
		})
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	result, err := repo.ListByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("ListByUser() returned %d grants, want 1", len(result))
	}
}

func TestAPIGrantRepoListByUserError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.ListByUser(context.Background(), uuid.New())
	if err == nil {
		t.Error("ListByUser() with 500 should return error")
	}
}

func TestAPIGrantRepoGetInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.Get(context.Background(), uuid.New())
	if err == nil {
		t.Error("Get() with invalid JSON should return error")
	}
}

func TestAPIGrantRepoListInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.List(context.Background())
	if err == nil {
		t.Error("List() with invalid JSON should return error")
	}
}

func TestAPIGrantRepoListByUserInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	repo := &APIGrantRepo{
		httpClient: server.Client(),
		authzURL:   server.URL,
	}

	_, err := repo.ListByUser(context.Background(), uuid.New())
	if err == nil {
		t.Error("ListByUser() with invalid JSON should return error")
	}
}
