package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
)

// DeviceCodeStatus represents the state of a device code.
type DeviceCodeStatus string

const (
	DeviceCodePending    DeviceCodeStatus = "PENDING"
	DeviceCodeAuthorized DeviceCodeStatus = "AUTHORIZED"
	DeviceCodeUsed       DeviceCodeStatus = "USED"
)

const (
	// DeviceCodePollInterval is the recommended poll interval for device code flow.
	DeviceCodePollInterval = 5 * time.Second
	// DeviceCodeTTL is the time-to-live for pending device codes.
	DeviceCodeTTL = 15 * time.Minute
	// UserCodeCharset defines allowed characters for user codes (no 0/O/1/I).
	UserCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	// UserCodeLength is the number of characters in a user code.
	UserCodeLength = 8
)

// DeviceCodeData holds the state of a device authorization.
type DeviceCodeData struct {
	DeviceCode string
	UserCode   string
	Status     DeviceCodeStatus
	UserID     string
	CreatedAt  time.Time
}

// DeviceCodeResponse is returned when a device code is generated.
type DeviceCodeResponse struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresIn       int // seconds
	Interval        int // seconds
}

// DeviceTokenResponse is returned when polling for a token.
type DeviceTokenResponse struct {
	AccessToken string
	TokenType   string
	Error       string
}

// DeviceAuthStore abstracts device code storage (Redis or in-memory).
// It is designed to be implemented by either the Redis adapter (server/internal/adapters/redis)
// for production use, or an in-memory store for testing.
type DeviceAuthStore interface {
	SaveDeviceCode(ctx context.Context, deviceCode string, data DeviceCodeData, ttl time.Duration) error
	GetDeviceCode(ctx context.Context, deviceCode string) (*DeviceCodeData, error)
	SaveUserCodeMapping(ctx context.Context, userCode string, deviceCode string, ttl time.Duration) error
	GetDeviceCodeByUserCode(ctx context.Context, userCode string) (string, error)
	ClaimDeviceCode(ctx context.Context, deviceCode string, ttl time.Duration) (bool, error)
	DeleteUserCodeMapping(ctx context.Context, userCode string) error
}

// DeviceAuthService handles the device authorization flow.
type DeviceAuthService struct {
	store           DeviceAuthStore
	tokenSvc        *ApiTokenService
	verificationURI string
}

// NewDeviceAuthService creates a new DeviceAuthService.
func NewDeviceAuthService(store DeviceAuthStore, tokenSvc *ApiTokenService, verificationURI string) *DeviceAuthService {
	if verificationURI == "" {
		verificationURI = "/device"
	}
	return &DeviceAuthService{
		store:           store,
		tokenSvc:        tokenSvc,
		verificationURI: verificationURI,
	}
}

// GenerateDeviceCode creates a new device authorization code pair.
func (s *DeviceAuthService) GenerateDeviceCode(ctx context.Context) (*DeviceCodeResponse, error) {
	// Generate device code: 32 random bytes, base64url.
	devBytes := make([]byte, 32)
	if _, err := rand.Read(devBytes); err != nil {
		return nil, fmt.Errorf("device: random device code: %w", err)
	}
	deviceCode := base64.RawURLEncoding.EncodeToString(devBytes)

	// Generate user code: 8 chars from charset, formatted XXXX-XXXX.
	userCode, err := generateUserCode()
	if err != nil {
		return nil, fmt.Errorf("device: generate user code: %w", err)
	}

	now := time.Now()
	data := DeviceCodeData{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		Status:     DeviceCodePending,
		CreatedAt:  now,
	}

	ttl := DeviceCodeTTL
	if err := s.store.SaveDeviceCode(ctx, deviceCode, data, ttl); err != nil {
		return nil, fmt.Errorf("device: save device code: %w", err)
	}
	if err := s.store.SaveUserCodeMapping(ctx, userCode, deviceCode, ttl); err != nil {
		return nil, fmt.Errorf("device: save user code mapping: %w", err)
	}

	return &DeviceCodeResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: s.verificationURI,
		ExpiresIn:       int(ttl.Seconds()),
		Interval:        5,
	}, nil
}

// AuthorizeDeviceCode authorizes a device code for a user.
func (s *DeviceAuthService) AuthorizeDeviceCode(ctx context.Context, userCode, userID string) error {
	deviceCode, err := s.store.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		return fmt.Errorf("device: invalid user code")
	}
	if deviceCode == "" {
		return fmt.Errorf("device: invalid user code")
	}

	data, err := s.store.GetDeviceCode(ctx, deviceCode)
	if err != nil {
		return fmt.Errorf("device: get device code: %w", err)
	}
	if data == nil {
		return fmt.Errorf("device: invalid device code")
	}

	switch data.Status {
	case DeviceCodePending:
		data.Status = DeviceCodeAuthorized
		data.UserID = userID
		return s.store.SaveDeviceCode(ctx, deviceCode, *data, DeviceCodeTTL)
	case DeviceCodeAuthorized:
		if data.UserID == userID {
			return nil // Already authorized by same user.
		}
		return fmt.Errorf("device: code already claimed")
	case DeviceCodeUsed:
		return fmt.Errorf("device: code already used")
	default:
		return fmt.Errorf("device: invalid code state")
	}
}

// PollToken polls for a completed device authorization.
func (s *DeviceAuthService) PollToken(ctx context.Context, deviceCode string) (*DeviceTokenResponse, error) {
	data, err := s.store.GetDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("device: get device code: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("device: invalid device code")
	}

	switch data.Status {
	case DeviceCodePending:
		return &DeviceTokenResponse{Error: "authorization_pending"}, nil
	case DeviceCodeAuthorized:
		// Try to atomically claim.
		claimed, err := s.store.ClaimDeviceCode(ctx, deviceCode, 1*time.Minute)
		if err != nil || !claimed {
			return &DeviceTokenResponse{Error: "authorization_pending"}, nil
		}
		defer s.store.DeleteUserCodeMapping(ctx, data.UserCode)

		// Create a CLI token.
		result, err := s.tokenSvc.RotateToken(ctx, data.UserID, "CLI Device Flow",
			[]string{ScopeSkillRead, ScopeSkillPublish}, nil)
		if err != nil {
			return nil, fmt.Errorf("device: create token: %w", err)
		}

		data.Status = DeviceCodeUsed
		_ = s.store.SaveDeviceCode(ctx, deviceCode, *data, 1*time.Minute)

		return &DeviceTokenResponse{
			AccessToken: result.RawToken,
			TokenType:   "Bearer",
		}, nil
	case DeviceCodeUsed:
		return nil, fmt.Errorf("device: code expired")
	default:
		return nil, fmt.Errorf("device: unknown state")
	}
}

// generateUserCode creates an 8-character human-readable code (XXXX-XXXX).
func generateUserCode() (string, error) {
	chars := []byte(UserCodeCharset)
	result := make([]byte, UserCodeLength)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[n.Int64()]
	}
	return string(result[0:4]) + "-" + string(result[4:8]), nil
}
