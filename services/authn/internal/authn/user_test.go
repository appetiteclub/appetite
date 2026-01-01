package authn

import (
	"testing"
	"time"

	authpkg "github.com/appetiteclub/apt/auth"
	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	user := NewUser()

	if user == nil {
		t.Fatal("NewUser() returned nil")
	}

	if user.ID == uuid.Nil {
		t.Error("NewUser() should generate a non-nil UUID")
	}

	if user.Status != authpkg.UserStatusActive {
		t.Errorf("NewUser() Status = %v, want %v", user.Status, authpkg.UserStatusActive)
	}
}

func TestUser_GetID(t *testing.T) {
	tests := []struct {
		name string
		user *User
		want uuid.UUID
	}{
		{
			name: "returns correct ID",
			user: &User{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
			want: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name: "returns nil UUID when not set",
			user: &User{},
			want: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.GetID(); got != tt.want {
				t.Errorf("User.GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_SetID(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "sets valid UUID",
			id:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		},
		{
			name: "sets nil UUID",
			id:   uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{}
			user.SetID(tt.id)

			if user.ID != tt.id {
				t.Errorf("User.SetID() ID = %v, want %v", user.ID, tt.id)
			}
		})
	}
}

func TestUser_ResourceType(t *testing.T) {
	user := &User{}
	got := user.ResourceType()
	want := "user"

	if got != want {
		t.Errorf("User.ResourceType() = %v, want %v", got, want)
	}
}

func TestUser_EnsureID(t *testing.T) {
	tests := []struct {
		name        string
		user        *User
		expectNewID bool
	}{
		{
			name:        "generates ID when nil",
			user:        &User{ID: uuid.Nil},
			expectNewID: true,
		},
		{
			name:        "preserves existing ID",
			user:        &User{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")},
			expectNewID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.user.ID
			tt.user.EnsureID()

			if tt.expectNewID {
				if tt.user.ID == uuid.Nil {
					t.Error("EnsureID() should generate non-nil UUID")
				}
			} else {
				if tt.user.ID != originalID {
					t.Errorf("EnsureID() changed existing ID from %v to %v", originalID, tt.user.ID)
				}
			}
		})
	}
}

func TestUser_BeforeCreate(t *testing.T) {
	user := &User{ID: uuid.Nil}
	beforeTime := time.Now()

	user.BeforeCreate()

	afterTime := time.Now()

	// Check ID was generated
	if user.ID == uuid.Nil {
		t.Error("BeforeCreate() should generate UUID")
	}

	// Check timestamps were set
	if user.CreatedAt.IsZero() {
		t.Error("BeforeCreate() should set CreatedAt")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("BeforeCreate() should set UpdatedAt")
	}

	// Check timestamps are reasonable
	if user.CreatedAt.Before(beforeTime) || user.CreatedAt.After(afterTime) {
		t.Error("BeforeCreate() CreatedAt timestamp is out of expected range")
	}
	if user.UpdatedAt.Before(beforeTime) || user.UpdatedAt.After(afterTime) {
		t.Error("BeforeCreate() UpdatedAt timestamp is out of expected range")
	}
}

func TestUser_BeforeUpdate(t *testing.T) {
	user := &User{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	originalCreatedAt := user.CreatedAt
	beforeTime := time.Now()

	user.BeforeUpdate()

	afterTime := time.Now()

	// Check CreatedAt was not changed
	if !user.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("BeforeUpdate() changed CreatedAt from %v to %v", originalCreatedAt, user.CreatedAt)
	}

	// Check UpdatedAt was updated
	if user.UpdatedAt.Before(beforeTime) || user.UpdatedAt.After(afterTime) {
		t.Error("BeforeUpdate() UpdatedAt timestamp is out of expected range")
	}
}

func TestUser_ToDomainUser(t *testing.T) {
	testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440004")
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	user := &User{
		ID:           testID,
		EmailCT:      []byte("encrypted_email"),
		EmailIV:      []byte("iv_data"),
		EmailTag:     []byte("tag_data"),
		EmailLookup:  []byte("lookup_hash"),
		PasswordHash: []byte("password_hash"),
		PasswordSalt: []byte("salt_data"),
		MFASecretCT:  []byte("mfa_secret"),
		Status:       authpkg.UserStatusActive,
		CreatedAt:    testTime,
	}

	domainUser := user.ToDomainUser()

	if domainUser == nil {
		t.Fatal("ToDomainUser() returned nil")
	}

	// Test all field mappings
	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ID", domainUser.ID, testID},
		{"EmailCT", string(domainUser.EmailCT), "encrypted_email"},
		{"EmailIV", string(domainUser.EmailIV), "iv_data"},
		{"EmailTag", string(domainUser.EmailTag), "tag_data"},
		{"EmailLookup", string(domainUser.EmailLookup), "lookup_hash"},
		{"PasswordHash", string(domainUser.PasswordHash), "password_hash"},
		{"PasswordSalt", string(domainUser.PasswordSalt), "salt_data"},
		{"MFASecretCT", string(domainUser.MFASecretCT), "mfa_secret"},
		{"Status", domainUser.Status, authpkg.UserStatusActive},
		{"CreatedAt", domainUser.CreatedAt, testTime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("ToDomainUser() %s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestFromDomainUser(t *testing.T) {
	testID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440005")
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	domainUser := &authpkg.User{
		ID:           testID,
		EmailCT:      []byte("encrypted_email"),
		EmailIV:      []byte("iv_data"),
		EmailTag:     []byte("tag_data"),
		EmailLookup:  []byte("lookup_hash"),
		PasswordHash: []byte("password_hash"),
		PasswordSalt: []byte("salt_data"),
		MFASecretCT:  []byte("mfa_secret"),
		Status:       authpkg.UserStatusSuspended,
		CreatedAt:    testTime,
	}

	user := FromDomainUser(domainUser)

	if user == nil {
		t.Fatal("FromDomainUser() returned nil")
	}

	// Test all field mappings
	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ID", user.ID, testID},
		{"EmailCT", string(user.EmailCT), "encrypted_email"},
		{"EmailIV", string(user.EmailIV), "iv_data"},
		{"EmailTag", string(user.EmailTag), "tag_data"},
		{"EmailLookup", string(user.EmailLookup), "lookup_hash"},
		{"PasswordHash", string(user.PasswordHash), "password_hash"},
		{"PasswordSalt", string(user.PasswordSalt), "salt_data"},
		{"MFASecretCT", string(user.MFASecretCT), "mfa_secret"},
		{"Status", user.Status, authpkg.UserStatusSuspended},
		{"CreatedAt", user.CreatedAt, testTime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("FromDomainUser() %s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}

	// Service-specific fields should be zero values
	if !user.UpdatedAt.IsZero() {
		t.Error("FromDomainUser() should leave UpdatedAt as zero value")
	}
	if user.CreatedBy != "" {
		t.Error("FromDomainUser() should leave CreatedBy as empty string")
	}
	if user.UpdatedBy != "" {
		t.Error("FromDomainUser() should leave UpdatedBy as empty string")
	}
}

func TestUser_RoundTripConversion(t *testing.T) {
	// Test that converting service User -> domain User -> service User preserves data
	original := &User{
		ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440006"),
		EmailCT:      []byte("test_email"),
		EmailIV:      []byte("test_iv"),
		EmailTag:     []byte("test_tag"),
		EmailLookup:  []byte("test_lookup"),
		PasswordHash: []byte("test_hash"),
		PasswordSalt: []byte("test_salt"),
		MFASecretCT:  []byte("test_mfa"),
		Status:       authpkg.UserStatusActive,
		CreatedAt:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		// Note: UpdatedAt, CreatedBy, UpdatedBy intentionally not set for this test
	}

	// Convert to domain and back
	domain := original.ToDomainUser()
	converted := FromDomainUser(domain)

	// Compare core fields (service-specific fields will be lost/reset)
	if converted.ID != original.ID {
		t.Errorf("Round trip ID = %v, want %v", converted.ID, original.ID)
	}
	if string(converted.EmailCT) != string(original.EmailCT) {
		t.Errorf("Round trip EmailCT = %v, want %v", converted.EmailCT, original.EmailCT)
	}
	if converted.Status != original.Status {
		t.Errorf("Round trip Status = %v, want %v", converted.Status, original.Status)
	}
	if !converted.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("Round trip CreatedAt = %v, want %v", converted.CreatedAt, original.CreatedAt)
	}
}
