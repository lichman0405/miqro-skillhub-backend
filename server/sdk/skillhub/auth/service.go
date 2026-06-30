package auth

// Service is the main auth SDK service, assembling all auth sub-services.
type Service struct {
	Local         *LocalAuthService
	Token         *ApiTokenService
	RBAC          *RbacService
	PasswordReset *PasswordResetService
	Device        *DeviceAuthService
	Merge         *AccountMergeService
	Policy        *RouteSecurityPolicyCatalog
}

// ServiceConfig holds the dependencies for creating an Auth Service.
type ServiceConfig struct {
	UserAccountRepo       UserAccountRepository
	LocalCredentialRepo   LocalCredentialRepository
	ApiTokenRepo          ApiTokenRepository
	RoleRepo              RoleRepository
	PermissionRepo        PermissionRepository
	UserRoleBindingRepo   UserRoleBindingRepository
	IdentityBindingRepo   IdentityBindingRepository
	PasswordResetRepo     PasswordResetRequestRepository
	AccountMergeRepo      AccountMergeRequestRepository
	DeviceStore           DeviceAuthStore
	VerificationURI       string
}

// NewService creates a fully wired Auth Service.
func NewService(cfg ServiceConfig) *Service {
	localAuth := NewLocalAuthService(
		cfg.UserAccountRepo,
		cfg.LocalCredentialRepo,
		cfg.UserRoleBindingRepo,
		cfg.RoleRepo,
	)

	tokenSvc := NewApiTokenService(cfg.ApiTokenRepo)

	rbac := NewRbacService(
		cfg.UserRoleBindingRepo,
		cfg.RoleRepo,
		cfg.PermissionRepo,
	)

	passwordReset := NewPasswordResetService(
		cfg.UserAccountRepo,
		cfg.LocalCredentialRepo,
		cfg.PasswordResetRepo,
	)

	var device *DeviceAuthService
	if cfg.DeviceStore != nil {
		device = NewDeviceAuthService(cfg.DeviceStore, tokenSvc, cfg.VerificationURI)
	}

	merge := NewAccountMergeService(
		cfg.UserAccountRepo,
		cfg.LocalCredentialRepo,
		cfg.IdentityBindingRepo,
		cfg.ApiTokenRepo,
		cfg.UserRoleBindingRepo,
		cfg.AccountMergeRepo,
	)

	policy := NewRouteSecurityPolicyCatalog()

	return &Service{
		Local:         localAuth,
		Token:         tokenSvc,
		RBAC:          rbac,
		PasswordReset: passwordReset,
		Device:        device,
		Merge:         merge,
		Policy:        policy,
	}
}
