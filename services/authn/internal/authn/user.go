package authn

import (
	"strings"
	"time"

	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
)

// User is the aggregate root for the User domain.
type User struct {
	ID           uuid.UUID          `json:"id" db:"id" bson:"_id"`
	Username     string             `json:"username" db:"username" bson:"username"`
	Name         string             `json:"name" db:"name" bson:"name"`
	EmailCT      []byte             `json:"-" db:"email_ct" bson:"email_ct"`
	EmailIV      []byte             `json:"-" db:"email_iv" bson:"email_iv"`
	EmailTag     []byte             `json:"-" db:"email_tag" bson:"email_tag"`
	EmailLookup  []byte             `json:"-" db:"email_lookup" bson:"email_lookup"`
	PasswordHash []byte             `json:"-" db:"password_hash" bson:"pass_hash"`
	PasswordSalt []byte             `json:"-" db:"password_salt" bson:"pass_salt"`
	MFASecretCT  []byte             `json:"-" db:"mfa_secret_ct" bson:"mfa_secret_ct,omitempty"`
	PINCT        []byte             `json:"-" db:"pin_ct" bson:"pin_ct,omitempty"`
	PINIV        []byte             `json:"-" db:"pin_iv" bson:"pin_iv,omitempty"`
	PINTag       []byte             `json:"-" db:"pin_tag" bson:"pin_tag,omitempty"`
	PINLookup    []byte             `json:"-" db:"pin_lookup" bson:"pin_lookup,omitempty"`
	Status       authpkg.UserStatus `json:"status" db:"status" bson:"status"`
	CreatedAt    time.Time          `json:"created_at" db:"created_at" bson:"created_at"`
	CreatedBy    string             `json:"created_by" db:"created_by" bson:"created_by"`
	UpdatedAt    time.Time          `json:"updated_at" db:"updated_at" bson:"updated_at"`
	UpdatedBy    string             `json:"updated_by" db:"updated_by" bson:"updated_by"`
}

// GetID returns the ID of the User (implements Identifiable interface).
func (u *User) GetID() uuid.UUID {
	return u.ID
}

// ResourceType returns the resource type for URL generation.
func (u *User) ResourceType() string {
	return "user"
}

// SetID sets the ID of the User.
func (u *User) SetID(id uuid.UUID) {
	u.ID = id
}

// NewUser creates a new User with a generated ID.
func NewUser() *User {
	return &User{
		ID:     aqm.GenerateNewID(),
		Status: authpkg.UserStatusActive,
	}
}

// EnsureID ensures the aggregate root has a valid ID.
func (u *User) EnsureID() {
	if u.ID == uuid.Nil {
		u.ID = aqm.GenerateNewID()
	}
}

// BeforeCreate sets creation timestamps.
func (u *User) BeforeCreate() {
	u.EnsureID()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	u.Username = normalizeUsernameField(u.Username)
	u.Name = strings.TrimSpace(u.Name)
}

// BeforeUpdate sets update timestamps.
func (u *User) BeforeUpdate() {
	u.UpdatedAt = time.Now()
	u.Username = normalizeUsernameField(u.Username)
	u.Name = strings.TrimSpace(u.Name)
}

// ToDomainUser converts service User to pure domain User for business logic.
func (u *User) ToDomainUser() *authpkg.User {
	return &authpkg.User{
		ID:           u.ID,
		Username:     u.Username,
		Name:         u.Name,
		EmailCT:      u.EmailCT,
		EmailIV:      u.EmailIV,
		EmailTag:     u.EmailTag,
		EmailLookup:  u.EmailLookup,
		PasswordHash: u.PasswordHash,
		PasswordSalt: u.PasswordSalt,
		MFASecretCT:  u.MFASecretCT,
		PINCT:        u.PINCT,
		PINIV:        u.PINIV,
		PINTag:       u.PINTag,
		PINLookup:    u.PINLookup,
		Status:       u.Status,
		CreatedAt:    u.CreatedAt,
	}
}

// FromDomainUser creates service User from pure domain User.
func FromDomainUser(domainUser *authpkg.User) *User {
	return &User{
		ID:           domainUser.ID,
		Username:     domainUser.Username,
		Name:         domainUser.Name,
		EmailCT:      domainUser.EmailCT,
		EmailIV:      domainUser.EmailIV,
		EmailTag:     domainUser.EmailTag,
		EmailLookup:  domainUser.EmailLookup,
		PasswordHash: domainUser.PasswordHash,
		PasswordSalt: domainUser.PasswordSalt,
		MFASecretCT:  domainUser.MFASecretCT,
		PINCT:        domainUser.PINCT,
		PINIV:        domainUser.PINIV,
		PINTag:       domainUser.PINTag,
		PINLookup:    domainUser.PINLookup,
		Status:       domainUser.Status,
		CreatedAt:    domainUser.CreatedAt,
		// Service-specific fields remain zero values initially
	}
}

func normalizeUsernameField(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	return strings.Trim(trimmed, "._-")
}
