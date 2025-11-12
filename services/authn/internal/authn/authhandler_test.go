package authn

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/appetiteclub/appetite/services/authn/internal/config"
	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
)

func setupAuthHandler() (*AuthHandler, *mockUserRepo) {
	repo := newMockUserRepo()
	log := aqm.NewNoopLogger()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			EncryptionKey:   "12345678901234567890123456789012", // 32 bytes
			SigningKey:      "signing-key-for-testing",
			SessionTTL:      "24h",
			TokenPrivateKey: "",
			TokenPublicKey:  "",
		},
	}

	xparams := config.NewXParams(log, cfg)

	handler := NewAuthHandler(repo, xparams)
	return handler, repo
}

func TestNewAuthHandler(t *testing.T) {
	handler, _ := setupAuthHandler()

	if handler == nil {
		t.Fatal("NewAuthHandler() returned nil")
	}

	if handler.repo == nil {
		t.Error("NewAuthHandler() handler.repo is nil")
	}
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	handler, _ := setupAuthHandler()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/authn/signup"},
		{http.MethodPost, "/authn/signin"},
		{http.MethodPost, "/authn/signout"},
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

func TestAuthHandlerSignUp(t *testing.T) {
	handler, repo := setupAuthHandler()

	tests := []struct {
		name           string
		body           string
		repoError      error
		existingUser   *User
		expectedStatus int
	}{
		{
			name:           "successful signup",
			body:           `{"email":"test@example.com","password":"ValidPassword123!","username":"testuser","name":"Test User"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid JSON",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation fails - invalid email",
			body:           `{"email":"invalid-email","password":"ValidPassword123!","username":"testuser","name":"Test User"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation fails - weak password",
			body:           `{"email":"test@example.com","password":"123","username":"testuser","name":"Test User"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "user already exists",
			body:           `{"email":"existing@example.com","password":"ValidPassword123!","username":"existinguser","name":"Existing User"}`,
			existingUser:   &User{ID: uuid.New(), Status: authpkg.UserStatusActive, Username: "existinguser"},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "repository error on lookup",
			body:           `{"email":"test@example.com","password":"ValidPassword123!","username":"testuser","name":"Test User"}`,
			repoError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "repository error on create",
			body:           `{"email":"test@example.com","password":"ValidPassword123!","username":"testuser","name":"Test User"}`,
			repoError:      fmt.Errorf("create error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset repository
			repo.users = make(map[uuid.UUID]*User)
			repo.createError = nil
			repo.getError = nil

			// Setup existing user if needed
			if tt.existingUser != nil {
				// Create lookup hash for existing user
				normalizedEmail := authpkg.NormalizeEmail("existing@example.com")
				signingKey := []byte(handler.cfg().Auth.SigningKey)
				emailLookup := authpkg.ComputeLookupHash(normalizedEmail, signingKey)
				tt.existingUser.EmailLookup = emailLookup
				repo.users[tt.existingUser.ID] = tt.existingUser
			}

			// Setup repository error
			if strings.Contains(tt.name, "repository error on lookup") {
				repo.getError = tt.repoError
			} else if strings.Contains(tt.name, "repository error on create") {
				repo.createError = tt.repoError
			}

			req := httptest.NewRequest(http.MethodPost, "/authn/signup", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.SignUp(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("SignUp() status = %d, want %d", rr.Code, tt.expectedStatus)
			}

			// Check user was created for successful case
			if tt.expectedStatus == http.StatusCreated {
				if len(repo.users) != 1 {
					t.Error("SignUp() should have created user")
				}
			}
		})
	}
}

func TestAuthHandlerSignIn(t *testing.T) {
	handler, repo := setupAuthHandler()

	// Create a valid test user
	validUserID := uuid.New()
	normalizedEmail := authpkg.NormalizeEmail("test@example.com")
	signingKey := []byte(handler.cfg().Auth.SigningKey)
	emailLookup := authpkg.ComputeLookupHash(normalizedEmail, signingKey)
	salt := authpkg.GeneratePasswordSalt()
	passwordHash := authpkg.HashPassword([]byte("ValidPassword123!"), salt)

	validUser := &User{
		ID:           validUserID,
		EmailLookup:  emailLookup,
		PasswordHash: passwordHash,
		PasswordSalt: salt,
		Status:       authpkg.UserStatusActive,
	}

	tests := []struct {
		name           string
		body           string
		setupUser      *User
		repoError      error
		expectedStatus int
	}{
		{
			name:           "successful signin",
			body:           `{"email":"test@example.com","password":"ValidPassword123!"}`,
			setupUser:      validUser,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation fails - empty email",
			body:           `{"email":"","password":"ValidPassword123!"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation fails - empty password",
			body:           `{"email":"test@example.com","password":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "user not found",
			body:           `{"email":"nonexistent@example.com","password":"ValidPassword123!"}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "wrong password",
			body:           `{"email":"test@example.com","password":"WrongPassword123!"}`,
			setupUser:      validUser,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user not active",
			body: `{"email":"test@example.com","password":"ValidPassword123!"}`,
			setupUser: &User{
				ID:           validUserID,
				EmailLookup:  emailLookup,
				PasswordHash: passwordHash,
				PasswordSalt: salt,
				Status:       authpkg.UserStatusSuspended,
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "repository error",
			body:           `{"email":"test@example.com","password":"ValidPassword123!"}`,
			repoError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset repository
			repo.users = make(map[uuid.UUID]*User)
			repo.getError = tt.repoError

			// Setup user if needed
			if tt.setupUser != nil {
				repo.users[tt.setupUser.ID] = tt.setupUser
			}

			req := httptest.NewRequest(http.MethodPost, "/authn/signin", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.SignIn(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("SignIn() status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAuthHandlerSignOut(t *testing.T) {
	handler, _ := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/authn/signout", nil)
	rr := httptest.NewRecorder()

	handler.SignOut(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("SignOut() status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestAuthHandlerDecodeSignUpPayload(t *testing.T) {
	handler, _ := setupAuthHandler()
	log := handler.log()

	tests := []struct {
		name     string
		body     string
		wantOK   bool
		wantCode int
		checkReq bool
	}{
		{
			name:     "valid JSON",
			body:     `{"email":"test@example.com","password":"ValidPassword123!"}`,
			wantOK:   true,
			checkReq: true,
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
			body:     strings.Repeat("a", AuthMaxBodyBytes+1),
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			if tt.name == "body too large" {
				req = httptest.NewRequest(http.MethodPost, "/authn/signup", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(http.MethodPost, "/authn/signup", bytes.NewBufferString(tt.body))
			}

			rr := httptest.NewRecorder()

			gotReq, gotOK := handler.decodeSignUpPayload(rr, req, log)

			if gotOK != tt.wantOK {
				t.Errorf("decodeSignUpPayload() gotOK = %v, want %v", gotOK, tt.wantOK)
			}

			if !tt.wantOK && rr.Code != tt.wantCode {
				t.Errorf("decodeSignUpPayload() status = %d, want %d", rr.Code, tt.wantCode)
			}

			if tt.checkReq && tt.wantOK {
				if gotReq.Email != "test@example.com" {
					t.Errorf("decodeSignUpPayload() Email = %s, want %s", gotReq.Email, "test@example.com")
				}
				if gotReq.Password != "ValidPassword123!" {
					t.Errorf("decodeSignUpPayload() Password = %s, want %s", gotReq.Password, "ValidPassword123!")
				}
			}
		})
	}
}

func TestAuthHandlerDecodeSignInPayload(t *testing.T) {
	handler, _ := setupAuthHandler()
	log := handler.log()

	tests := []struct {
		name     string
		body     string
		wantOK   bool
		wantCode int
		checkReq bool
	}{
		{
			name:     "valid JSON",
			body:     `{"email":"test@example.com","password":"password123"}`,
			wantOK:   true,
			checkReq: true,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/authn/signin", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()

			gotReq, gotOK := handler.decodeSignInPayload(rr, req, log)

			if gotOK != tt.wantOK {
				t.Errorf("decodeSignInPayload() gotOK = %v, want %v", gotOK, tt.wantOK)
			}

			if !tt.wantOK && rr.Code != tt.wantCode {
				t.Errorf("decodeSignInPayload() status = %d, want %d", rr.Code, tt.wantCode)
			}

			if tt.checkReq && tt.wantOK {
				if gotReq.Email != "test@example.com" {
					t.Errorf("decodeSignInPayload() Email = %s, want %s", gotReq.Email, "test@example.com")
				}
				if gotReq.Password != "password123" {
					t.Errorf("decodeSignInPayload() Password = %s, want %s", gotReq.Password, "password123")
				}
			}
		})
	}
}

func TestAuthHandlerGenerateSessionToken(t *testing.T) {
	handler, _ := setupAuthHandler()

	tests := []struct {
		name        string
		userID      string
		sessionTTL  string
		tokenKey    string
		expectError bool
	}{
		{
			name:        "valid token generation",
			userID:      uuid.New().String(),
			sessionTTL:  "24h",
			expectError: false,
		},
		{
			name:        "invalid session TTL",
			userID:      uuid.New().String(),
			sessionTTL:  "invalid-duration",
			expectError: true,
		},
		{
			name:        "with configured private key",
			userID:      uuid.New().String(),
			sessionTTL:  "1h",
			tokenKey:    generateBase64Ed25519Key(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config for this test
			handler.cfg().Auth.SessionTTL = tt.sessionTTL
			handler.cfg().Auth.TokenPrivateKey = tt.tokenKey

			userID, parseErr := uuid.Parse(tt.userID)
			if parseErr != nil {
				t.Fatalf("invalid test user id: %v", parseErr)
			}

			token, err := generateSessionToken(handler.cfg(), userID)

			if tt.expectError {
				if err == nil {
					t.Error("generateSessionToken() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("generateSessionToken() unexpected error: %v", err)
				}
				if token == "" {
					t.Error("generateSessionToken() returned empty token")
				}
			}
		})
	}
}

func TestAuthHandlerGetTokenPrivateKey(t *testing.T) {
	handler, _ := setupAuthHandler()

	tests := []struct {
		name                string
		configuredKey       string
		expectError         bool
		expectKeyGeneration bool
	}{
		{
			name:          "valid configured key",
			configuredKey: generateBase64Ed25519Key(),
			expectError:   false,
		},
		{
			name:          "invalid configured key",
			configuredKey: "invalid-key",
			expectError:   true,
		},
		{
			name:                "no configured key - generate new",
			configuredKey:       "",
			expectError:         false,
			expectKeyGeneration: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler.cfg().Auth.TokenPrivateKey = tt.configuredKey

			privateKey, err := tokenPrivateKey(handler.cfg().Auth.TokenPrivateKey)

			if tt.expectError {
				if err == nil {
					t.Error("getTokenPrivateKey() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("getTokenPrivateKey() unexpected error: %v", err)
				}
				if privateKey == nil {
					t.Error("getTokenPrivateKey() returned nil private key")
				}
				if len(privateKey) == 0 {
					t.Error("getTokenPrivateKey() returned empty private key")
				}
			}
		})
	}
}

func TestAuthHandlerLog(t *testing.T) {
	handler, _ := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/authn/signin", nil)
	log := handler.log(req)

	if log == nil {
		t.Error("log() returned nil logger")
	}
}

// Helper function to generate a valid base64 encoded Ed25519 private key for testing
func generateBase64Ed25519Key() string {
	_, privateKey, _ := authpkg.GenerateKeyPair()
	return base64.StdEncoding.EncodeToString(privateKey)
}

// Test request/response structs
func TestSignUpRequest(t *testing.T) {
	req := SignUpRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	if req.Email != "test@example.com" {
		t.Errorf("SignUpRequest.Email = %s, want %s", req.Email, "test@example.com")
	}
	if req.Password != "password123" {
		t.Errorf("SignUpRequest.Password = %s, want %s", req.Password, "password123")
	}
}

func TestSignInRequest(t *testing.T) {
	req := SignInRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	if req.Email != "test@example.com" {
		t.Errorf("SignInRequest.Email = %s, want %s", req.Email, "test@example.com")
	}
	if req.Password != "password123" {
		t.Errorf("SignInRequest.Password = %s, want %s", req.Password, "password123")
	}
}

func TestAuthResponse(t *testing.T) {
	user := &User{ID: uuid.New(), Status: authpkg.UserStatusActive}
	token := "test-token"

	resp := AuthResponse{
		User:  user,
		Token: token,
	}

	if resp.User != user {
		t.Error("AuthResponse.User mismatch")
	}
	if resp.Token != token {
		t.Errorf("AuthResponse.Token = %s, want %s", resp.Token, token)
	}
}
