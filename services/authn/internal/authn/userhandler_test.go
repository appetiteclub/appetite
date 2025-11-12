package authn

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/appetiteclub/appetite/services/authn/internal/config"
	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
)

type mockUserRepo struct {
	users       map[uuid.UUID]*User
	createError error
	getError    error
	saveError   error
	deleteError error
	listError   error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[uuid.UUID]*User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *User) error {
	if m.createError != nil {
		return m.createError
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	user, exists := m.users[id]
	if !exists {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepo) GetByEmailLookup(ctx context.Context, lookup []byte) (*User, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	for _, user := range m.users {
		if bytes.Equal(user.EmailLookup, lookup) {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) Save(ctx context.Context, user *User) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) List(ctx context.Context) ([]*User, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	var users []*User
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *mockUserRepo) ListByStatus(ctx context.Context, status string) ([]*User, error) {
	var users []*User
	for _, user := range m.users {
		if string(user.Status) == status {
			users = append(users, user)
		}
	}
	return users, nil
}

func setupUserHandler() (*UserHandler, *mockUserRepo) {
	repo := newMockUserRepo()
	log := aqm.NewNoopLogger()
	xparams := config.NewXParams(log, &config.Config{})
	handler := NewUserHandler(repo, xparams)
	return handler, repo
}

func TestNewUserHandler(t *testing.T) {
	handler, _ := setupUserHandler()

	if handler == nil {
		t.Fatal("NewUserHandler() returned nil")
	}

	if handler.repo == nil {
		t.Error("NewUserHandler() handler.repo is nil")
	}
}

// TestUserHandler_CreateUser_Direct tests the core handler logic by directly calling with valid users
func TestUserHandlerCreateUserDirect(t *testing.T) {
	handler, repo := setupUserHandler()

	// Create a valid user that would pass validation
	validUser := User{
		EmailLookup:  []byte("test@example.com"),
		PasswordHash: []byte("hashed"),
		PasswordSalt: []byte("salt"),
		Status:       authpkg.UserStatusActive,
	}

	tests := []struct {
		name          string
		user          User
		repoError     error
		expectedUsers int
	}{
		{
			name:          "successful creation",
			user:          validUser,
			expectedUsers: 1,
		},
		{
			name:          "repository error",
			user:          validUser,
			repoError:     fmt.Errorf("database error"),
			expectedUsers: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset
			repo.users = make(map[uuid.UUID]*User)
			repo.createError = tt.repoError

			// Prepare user as handler would
			user := tt.user
			user.EnsureID()
			user.BeforeCreate()

			// Test the core repository interaction
			err := handler.repo.Create(context.Background(), &user)

			if tt.repoError != nil {
				if err == nil {
					t.Error("Expected repository error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if len(repo.users) != tt.expectedUsers {
				t.Errorf("Expected %d users, got %d", tt.expectedUsers, len(repo.users))
			}
		})
	}
}

func TestUserHandlerCreateUser(t *testing.T) {
	handler, repo := setupUserHandler()

	tests := []struct {
		name           string
		body           string
		repoError      error
		expectedStatus int
		expectUser     bool
	}{
		{
			name:           "invalid JSON",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectUser:     false,
		},
		{
			name:           "empty body",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectUser:     false,
		},
		{
			name:           "validation fails - missing required fields",
			body:           `{"status":"active"}`,
			expectedStatus: http.StatusBadRequest,
			expectUser:     false,
		},
		{
			name:           "repository error",
			body:           `{"name":"Test User","email":"test@example.com","password":"password123","status":"active","username":"testuser"}`,
			repoError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectUser:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset errors
			repo.createError = nil
			repo.getError = nil
			repo.saveError = nil
			repo.deleteError = nil
			repo.listError = nil

			repo.createError = tt.repoError

			req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.CreateUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("CreateUser() status = %d, want %d", rr.Code, tt.expectedStatus)
			}

			if tt.expectUser && tt.expectedStatus == http.StatusCreated {
				if len(repo.users) == 0 {
					t.Error("CreateUser() should have created user in repository")
				}
			}
		})
	}
}

func TestUserHandlerGetUser(t *testing.T) {
	handler, repo := setupUserHandler()

	existingID := uuid.New()
	existingUser := &User{
		ID:           existingID,
		EmailLookup:  []byte("test@example.com"),
		PasswordHash: []byte("hashed"),
		PasswordSalt: []byte("salt"),
		Status:       authpkg.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.users[existingID] = existingUser

	tests := []struct {
		name           string
		userID         string
		repoError      error
		expectedStatus int
	}{
		{
			name:           "successful get",
			userID:         existingID.String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user not found",
			userID:         uuid.New().String(),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid UUID",
			userID:         "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty UUID",
			userID:         "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "repository error",
			userID:         existingID.String(),
			repoError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset errors
			repo.createError = nil
			repo.getError = nil
			repo.saveError = nil
			repo.deleteError = nil
			repo.listError = nil

			repo.getError = tt.repoError

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			handler.GetUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("GetUser() status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestUserHandlerGetAllUsers(t *testing.T) {
	handler, repo := setupUserHandler()

	user1 := &User{ID: uuid.New(), Status: authpkg.UserStatusActive}
	user2 := &User{ID: uuid.New(), Status: authpkg.UserStatusActive}
	repo.users[user1.ID] = user1
	repo.users[user2.ID] = user2

	tests := []struct {
		name           string
		repoError      error
		expectedStatus int
	}{
		{
			name:           "successful list",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "repository error",
			repoError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset errors
			repo.createError = nil
			repo.getError = nil
			repo.saveError = nil
			repo.deleteError = nil
			repo.listError = nil

			repo.listError = tt.repoError

			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			rr := httptest.NewRecorder()

			handler.GetAllUsers(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("GetAllUsers() status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestUserHandlerUpdateUser(t *testing.T) {
	handler, repo := setupUserHandler()

	existingID := uuid.New()
	existingUser := &User{
		ID:           existingID,
		EmailLookup:  []byte("test@example.com"),
		PasswordHash: []byte("hashed"),
		PasswordSalt: []byte("salt"),
		Status:       authpkg.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.users[existingID] = existingUser

	tests := []struct {
		name           string
		userID         string
		body           string
		repoError      error
		expectedStatus int
	}{
		{
			name:           "successful update",
			userID:         existingID.String(),
			body:           `{"name":"Updated User","email":"updated@example.com","status":"active","username":"updated-user"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid UUID",
			userID:         "invalid-uuid",
			body:           `{"status":"active"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON payload",
			userID:         existingID.String(),
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			userID:         existingID.String(),
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "repository error",
			userID:         existingID.String(),
			body:           `{"name":"Updated User","email":"updated@example.com","status":"active","username":"updated-user"}`,
			repoError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset errors
			repo.createError = nil
			repo.getError = nil
			repo.saveError = nil
			repo.deleteError = nil
			repo.listError = nil

			repo.saveError = tt.repoError

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			handler.UpdateUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("UpdateUser() status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestUserHandlerDeleteUser(t *testing.T) {
	handler, repo := setupUserHandler()

	existingID := uuid.New()
	existingUser := &User{ID: existingID, Status: authpkg.UserStatusActive}
	repo.users[existingID] = existingUser

	tests := []struct {
		name           string
		userID         string
		repoError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			userID:         existingID.String(),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid UUID",
			userID:         "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty UUID",
			userID:         "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "repository error",
			userID:         existingID.String(),
			repoError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset errors
			repo.createError = nil
			repo.getError = nil
			repo.saveError = nil
			repo.deleteError = nil
			repo.listError = nil

			repo.deleteError = tt.repoError

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.userID, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			handler.DeleteUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("DeleteUser() status = %d, want %d", rr.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusNoContent {
				if _, exists := repo.users[existingID]; exists {
					t.Error("DeleteUser() should have removed user from repository")
				}
			}
		})
	}
}

func TestUserHandlerParseIDParam(t *testing.T) {
	handler, _ := setupUserHandler()

	validUUID := uuid.New()

	tests := []struct {
		name     string
		idParam  string
		wantID   uuid.UUID
		wantOK   bool
		wantCode int
	}{
		{
			name:    "valid UUID",
			idParam: validUUID.String(),
			wantID:  validUUID,
			wantOK:  true,
		},
		{
			name:     "empty UUID",
			idParam:  "",
			wantID:   uuid.Nil,
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid UUID format",
			idParam:  "invalid-uuid",
			wantID:   uuid.Nil,
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "whitespace UUID",
			idParam:  "whitespace-id",
			wantID:   uuid.Nil,
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.idParam, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.idParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			gotID, gotOK := handler.parseIDParam(rr, req)

			if gotID != tt.wantID {
				t.Errorf("parseIDParam() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotOK != tt.wantOK {
				t.Errorf("parseIDParam() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
			if !tt.wantOK && rr.Code != tt.wantCode {
				t.Errorf("parseIDParam() status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}

func TestUserHandlerDecodeUserPayload(t *testing.T) {
	handler, _ := setupUserHandler()

	tests := []struct {
		name      string
		body      string
		wantOK    bool
		wantCode  int
		checkUser bool
	}{
		{
			name:      "valid JSON",
			body:      `{"status":"active"}`,
			wantOK:    true,
			checkUser: true,
		},
		{
			name:     "invalid JSON",
			body:     `{"invalid": json}`,
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty body",
			body:     "",
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "whitespace only body",
			body:     "   \n\t  ",
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "body too large",
			body:     strings.Repeat("a", UserMaxBodyBytes+1),
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			if tt.name == "body too large" {
				req = httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(tt.body))
			}

			rr := httptest.NewRecorder()

			gotUser, gotOK := handler.decodeUserCreatePayload(rr, req)

			if gotOK != tt.wantOK {
				t.Errorf("decodeUserPayload() gotOK = %v, want %v", gotOK, tt.wantOK)
			}

			if !tt.wantOK && rr.Code != tt.wantCode {
				t.Errorf("decodeUserPayload() status = %d, want %d", rr.Code, tt.wantCode)
			}

			if tt.checkUser && tt.wantOK {
				if gotUser.Status != "active" {
					t.Errorf("decodeUserPayload() Status = %s, want %s", gotUser.Status, "active")
				}
			}
		})
	}
}

func TestUserHandler_log(t *testing.T) {
	handler, _ := setupUserHandler()

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	log := handler.log(req)

	if log == nil {
		t.Error("log() returned nil logger")
	}
}

func TestUserHandlerRegisterRoutes(t *testing.T) {
	handler, _ := setupUserHandler()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/users/"},
		{http.MethodGet, "/users/"},
		{http.MethodGet, "/users/123"},
		{http.MethodPut, "/users/123"},
		{http.MethodDelete, "/users/123"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.path), func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			if rr.Code == http.StatusNotFound {
				t.Errorf("Route %s %s not registered", tt.method, tt.path)
			}
		})
	}
}
