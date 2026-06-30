package auth

import "context"

// DirectAuthRequest holds credentials for direct authentication.
type DirectAuthRequest struct {
	Username string
	Password string
}

// DirectAuthProvider is the interface for direct authentication providers.
type DirectAuthProvider interface {
	ProviderCode() string
	DisplayName() string
	Authenticate(ctx context.Context, req DirectAuthRequest) (*PlatformPrincipal, error)
}

// LocalDirectAuthProvider implements DirectAuthProvider using LocalAuthService.
type LocalDirectAuthProvider struct {
	localAuth *LocalAuthService
}

// NewLocalDirectAuthProvider creates a new LocalDirectAuthProvider.
func NewLocalDirectAuthProvider(localAuth *LocalAuthService) *LocalDirectAuthProvider {
	return &LocalDirectAuthProvider{localAuth: localAuth}
}

// ProviderCode returns "local".
func (p *LocalDirectAuthProvider) ProviderCode() string { return "local" }

// DisplayName returns "Local Account".
func (p *LocalDirectAuthProvider) DisplayName() string { return "Local Account" }

// Authenticate delegates to LocalAuthService.Login.
func (p *LocalDirectAuthProvider) Authenticate(ctx context.Context, req DirectAuthRequest) (*PlatformPrincipal, error) {
	return p.localAuth.Login(ctx, req.Username, req.Password)
}
