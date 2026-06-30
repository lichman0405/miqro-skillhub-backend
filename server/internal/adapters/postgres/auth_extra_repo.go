package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/auth"
)

// ApiTokenRepo implements auth.ApiTokenRepository.
type ApiTokenRepo struct{ *DB }

func NewApiTokenRepo(db *DB) *ApiTokenRepo { return &ApiTokenRepo{DB: db} }

func (r *ApiTokenRepo) Save(ctx context.Context, t auth.ApiToken) (auth.ApiToken, error) {
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}

	err := r.queryRow(ctx,
		`INSERT INTO api_token (subject_type, subject_id, user_id, name, token_prefix, token_hash, scope_json, expires_at, last_used_at, revoked_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (id) DO UPDATE SET
		   subject_type = EXCLUDED.subject_type,
		   subject_id = EXCLUDED.subject_id,
		   user_id = EXCLUDED.user_id,
		   name = EXCLUDED.name,
		   token_prefix = EXCLUDED.token_prefix,
		   token_hash = EXCLUDED.token_hash,
		   scope_json = EXCLUDED.scope_json,
		   expires_at = EXCLUDED.expires_at,
		   last_used_at = EXCLUDED.last_used_at,
		   revoked_at = EXCLUDED.revoked_at
		 RETURNING id, subject_type, subject_id, user_id, name, token_prefix, token_hash, scope_json, expires_at, last_used_at, revoked_at, created_at`,
		t.SubjectType, t.SubjectID, t.UserID, t.Name, t.TokenPrefix, t.TokenHash, t.ScopeJSON,
		t.ExpiresAt, t.LastUsedAt, t.RevokedAt, t.CreatedAt,
	).Scan(&t.ID, &t.SubjectType, &t.SubjectID, &t.UserID, &t.Name, &t.TokenPrefix, &t.TokenHash,
		&t.ScopeJSON, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		return auth.ApiToken{}, err
	}
	return t, nil
}

func (r *ApiTokenRepo) FindByID(ctx context.Context, id int64) (*auth.ApiToken, error) {
	var t auth.ApiToken
	err := r.queryRow(ctx,
		`SELECT id, subject_type, subject_id, user_id, name, token_prefix, token_hash, scope_json, expires_at, last_used_at, revoked_at, created_at
		 FROM api_token WHERE id = $1`, id,
	).Scan(&t.ID, &t.SubjectType, &t.SubjectID, &t.UserID, &t.Name, &t.TokenPrefix, &t.TokenHash,
		&t.ScopeJSON, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ApiTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*auth.ApiToken, error) {
	var t auth.ApiToken
	err := r.queryRow(ctx,
		`SELECT id, subject_type, subject_id, user_id, name, token_prefix, token_hash, scope_json, expires_at, last_used_at, revoked_at, created_at
		 FROM api_token WHERE token_hash = $1`, hash,
	).Scan(&t.ID, &t.SubjectType, &t.SubjectID, &t.UserID, &t.Name, &t.TokenPrefix, &t.TokenHash,
		&t.ScopeJSON, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ApiTokenRepo) FindByUserID(ctx context.Context, userID string) ([]auth.ApiToken, error) {
	rows, err := r.query(ctx,
		`SELECT id, subject_type, subject_id, user_id, name, token_prefix, token_hash, scope_json, expires_at, last_used_at, revoked_at, created_at
		 FROM api_token WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []auth.ApiToken
	for rows.Next() {
		var t auth.ApiToken
		if err := rows.Scan(&t.ID, &t.SubjectType, &t.SubjectID, &t.UserID, &t.Name, &t.TokenPrefix, &t.TokenHash,
			&t.ScopeJSON, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (r *ApiTokenRepo) FindActiveByName(ctx context.Context, userID string, name string) (*auth.ApiToken, error) {
	var t auth.ApiToken
	err := r.queryRow(ctx,
		`SELECT id, subject_type, subject_id, user_id, name, token_prefix, token_hash, scope_json, expires_at, last_used_at, revoked_at, created_at
		 FROM api_token WHERE user_id = $1 AND LOWER(name) = LOWER($2) AND revoked_at IS NULL`, userID, name,
	).Scan(&t.ID, &t.SubjectType, &t.SubjectID, &t.UserID, &t.Name, &t.TokenPrefix, &t.TokenHash,
		&t.ScopeJSON, &t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ApiTokenRepo) UpdateLastUsed(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `UPDATE api_token SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *ApiTokenRepo) Revoke(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `UPDATE api_token SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

// RoleRepo implements auth.RoleRepository.
type RoleRepo struct{ *DB }

func NewRoleRepo(db *DB) *RoleRepo { return &RoleRepo{DB: db} }

func (r *RoleRepo) FindByID(ctx context.Context, id int64) (*auth.Role, error) {
	var ro auth.Role
	err := r.queryRow(ctx,
		`SELECT id, code, name, description, is_system, created_at FROM role WHERE id = $1`, id,
	).Scan(&ro.ID, &ro.Code, &ro.Name, &ro.Description, &ro.IsSystem, &ro.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ro, nil
}

func (r *RoleRepo) FindByCode(ctx context.Context, code string) (*auth.Role, error) {
	var ro auth.Role
	err := r.queryRow(ctx,
		`SELECT id, code, name, description, is_system, created_at FROM role WHERE code = $1`, code,
	).Scan(&ro.ID, &ro.Code, &ro.Name, &ro.Description, &ro.IsSystem, &ro.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ro, nil
}

func (r *RoleRepo) FindAll(ctx context.Context) ([]auth.Role, error) {
	rows, err := r.query(ctx,
		`SELECT id, code, name, description, is_system, created_at FROM role ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []auth.Role
	for rows.Next() {
		var ro auth.Role
		if err := rows.Scan(&ro.ID, &ro.Code, &ro.Name, &ro.Description, &ro.IsSystem, &ro.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, ro)
	}
	return roles, rows.Err()
}

func (r *RoleRepo) Save(ctx context.Context, ro auth.Role) (auth.Role, error) {
	err := r.queryRow(ctx,
		`INSERT INTO role (code, name, description, is_system, created_at)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (code) DO UPDATE SET
		   name = EXCLUDED.name,
		   description = EXCLUDED.description,
		   is_system = EXCLUDED.is_system
		 RETURNING id, code, name, description, is_system, created_at`,
		ro.Code, ro.Name, ro.Description, ro.IsSystem, ro.CreatedAt,
	).Scan(&ro.ID, &ro.Code, &ro.Name, &ro.Description, &ro.IsSystem, &ro.CreatedAt)
	if err != nil {
		return auth.Role{}, err
	}
	return ro, nil
}

// PermissionRepo implements auth.PermissionRepository.
type PermissionRepo struct{ *DB }

func NewPermissionRepo(db *DB) *PermissionRepo { return &PermissionRepo{DB: db} }

func (r *PermissionRepo) FindByID(ctx context.Context, id int64) (*auth.Permission, error) {
	var p auth.Permission
	err := r.queryRow(ctx,
		`SELECT id, code, name, group_code FROM permission WHERE id = $1`, id,
	).Scan(&p.ID, &p.Code, &p.Name, &p.GroupCode)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PermissionRepo) FindByCode(ctx context.Context, code string) (*auth.Permission, error) {
	var p auth.Permission
	err := r.queryRow(ctx,
		`SELECT id, code, name, group_code FROM permission WHERE code = $1`, code,
	).Scan(&p.ID, &p.Code, &p.Name, &p.GroupCode)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PermissionRepo) FindAll(ctx context.Context) ([]auth.Permission, error) {
	rows, err := r.query(ctx,
		`SELECT id, code, name, group_code FROM permission ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []auth.Permission
	for rows.Next() {
		var p auth.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.GroupCode); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// UserRoleBindingRepo implements auth.UserRoleBindingRepository.
type UserRoleBindingRepo struct{ *DB }

func NewUserRoleBindingRepo(db *DB) *UserRoleBindingRepo { return &UserRoleBindingRepo{DB: db} }

func (r *UserRoleBindingRepo) Save(ctx context.Context, b auth.UserRoleBinding) (auth.UserRoleBinding, error) {
	err := r.queryRow(ctx,
		`INSERT INTO user_role_binding (user_id, role_id, created_at)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (user_id, role_id) DO UPDATE SET
		   created_at = EXCLUDED.created_at
		 RETURNING id, user_id, role_id, created_at`,
		b.UserID, b.RoleID, b.CreatedAt,
	).Scan(&b.ID, &b.UserID, &b.RoleID, &b.CreatedAt)
	if err != nil {
		return auth.UserRoleBinding{}, err
	}
	return b, nil
}

func (r *UserRoleBindingRepo) FindByUserID(ctx context.Context, userID string) ([]auth.UserRoleBinding, error) {
	rows, err := r.query(ctx,
		`SELECT id, user_id, role_id, created_at FROM user_role_binding WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []auth.UserRoleBinding
	for rows.Next() {
		var b auth.UserRoleBinding
		if err := rows.Scan(&b.ID, &b.UserID, &b.RoleID, &b.CreatedAt); err != nil {
			return nil, err
		}
		bindings = append(bindings, b)
	}
	return bindings, rows.Err()
}

func (r *UserRoleBindingRepo) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.exec(ctx, `DELETE FROM user_role_binding WHERE user_id = $1`, userID)
	return err
}

// IdentityBindingRepo implements auth.IdentityBindingRepository.
type IdentityBindingRepo struct{ *DB }

func NewIdentityBindingRepo(db *DB) *IdentityBindingRepo { return &IdentityBindingRepo{DB: db} }

func (r *IdentityBindingRepo) Save(ctx context.Context, b auth.IdentityBinding) (auth.IdentityBinding, error) {
	now := time.Now()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now

	err := r.queryRow(ctx,
		`INSERT INTO identity_binding (user_id, provider_code, subject, login_name, extra_json, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (provider_code, subject) DO UPDATE SET
		   user_id = EXCLUDED.user_id,
		   login_name = EXCLUDED.login_name,
		   extra_json = EXCLUDED.extra_json,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, user_id, provider_code, subject, login_name, extra_json, created_at, updated_at`,
		b.UserID, b.ProviderCode, b.Subject, b.LoginName, b.ExtraJSON, b.CreatedAt, b.UpdatedAt,
	).Scan(&b.ID, &b.UserID, &b.ProviderCode, &b.Subject, &b.LoginName, &b.ExtraJSON, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return auth.IdentityBinding{}, err
	}
	return b, nil
}

func (r *IdentityBindingRepo) FindByProviderAndSubject(ctx context.Context, providerCode string, subject string) (*auth.IdentityBinding, error) {
	var b auth.IdentityBinding
	err := r.queryRow(ctx,
		`SELECT id, user_id, provider_code, subject, login_name, extra_json, created_at, updated_at
		 FROM identity_binding WHERE provider_code = $1 AND subject = $2`, providerCode, subject,
	).Scan(&b.ID, &b.UserID, &b.ProviderCode, &b.Subject, &b.LoginName, &b.ExtraJSON, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *IdentityBindingRepo) FindByUserID(ctx context.Context, userID string) ([]auth.IdentityBinding, error) {
	rows, err := r.query(ctx,
		`SELECT id, user_id, provider_code, subject, login_name, extra_json, created_at, updated_at
		 FROM identity_binding WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []auth.IdentityBinding
	for rows.Next() {
		var b auth.IdentityBinding
		if err := rows.Scan(&b.ID, &b.UserID, &b.ProviderCode, &b.Subject, &b.LoginName, &b.ExtraJSON, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bindings = append(bindings, b)
	}
	return bindings, rows.Err()
}

// LocalCredentialRepo implements auth.LocalCredentialRepository.
type LocalCredentialRepo struct{ *DB }

func NewLocalCredentialRepo(db *DB) *LocalCredentialRepo { return &LocalCredentialRepo{DB: db} }

func (r *LocalCredentialRepo) Save(ctx context.Context, c auth.LocalCredential) (auth.LocalCredential, error) {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	err := r.queryRow(ctx,
		`INSERT INTO local_credential (user_id, username, password_hash, failed_attempts, locked_until, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (user_id) DO UPDATE SET
		   username = EXCLUDED.username,
		   password_hash = EXCLUDED.password_hash,
		   failed_attempts = EXCLUDED.failed_attempts,
		   locked_until = EXCLUDED.locked_until,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, user_id, username, password_hash, failed_attempts, locked_until, created_at, updated_at`,
		c.UserID, c.Username, c.PasswordHash, c.FailedAttempts, c.LockedUntil, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID, &c.UserID, &c.Username, &c.PasswordHash, &c.FailedAttempts, &c.LockedUntil, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return auth.LocalCredential{}, err
	}
	return c, nil
}

func (r *LocalCredentialRepo) FindByUserID(ctx context.Context, userID string) (*auth.LocalCredential, error) {
	var c auth.LocalCredential
	err := r.queryRow(ctx,
		`SELECT id, user_id, username, password_hash, failed_attempts, locked_until, created_at, updated_at
		 FROM local_credential WHERE user_id = $1`, userID,
	).Scan(&c.ID, &c.UserID, &c.Username, &c.PasswordHash, &c.FailedAttempts, &c.LockedUntil, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *LocalCredentialRepo) FindByUsername(ctx context.Context, username string) (*auth.LocalCredential, error) {
	var c auth.LocalCredential
	err := r.queryRow(ctx,
		`SELECT id, user_id, username, password_hash, failed_attempts, locked_until, created_at, updated_at
		 FROM local_credential WHERE username = $1`, username,
	).Scan(&c.ID, &c.UserID, &c.Username, &c.PasswordHash, &c.FailedAttempts, &c.LockedUntil, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// PasswordResetRequestRepo implements auth.PasswordResetRequestRepository.
type PasswordResetRequestRepo struct{ *DB }

func NewPasswordResetRequestRepo(db *DB) *PasswordResetRequestRepo {
	return &PasswordResetRequestRepo{DB: db}
}

func (r *PasswordResetRequestRepo) Save(ctx context.Context, req auth.PasswordResetRequest) (auth.PasswordResetRequest, error) {
	err := r.queryRow(ctx,
		`INSERT INTO password_reset_request (user_id, email, code_hash, expires_at, consumed_at, requested_by_admin, requested_by_user_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, user_id, email, code_hash, expires_at, consumed_at, requested_by_admin, requested_by_user_id, created_at`,
		req.UserID, req.Email, req.CodeHash, req.ExpiresAt, req.ConsumedAt, req.RequestedByAdmin, req.RequestedByUserID, req.CreatedAt,
	).Scan(&req.ID, &req.UserID, &req.Email, &req.CodeHash, &req.ExpiresAt, &req.ConsumedAt,
		&req.RequestedByAdmin, &req.RequestedByUserID, &req.CreatedAt)
	if err != nil {
		return auth.PasswordResetRequest{}, err
	}
	return req, nil
}

func (r *PasswordResetRequestRepo) FindValidByUserID(ctx context.Context, userID string) ([]auth.PasswordResetRequest, error) {
	rows, err := r.query(ctx,
		`SELECT id, user_id, email, code_hash, expires_at, consumed_at, requested_by_admin, requested_by_user_id, created_at
		 FROM password_reset_request WHERE user_id = $1 AND consumed_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []auth.PasswordResetRequest
	for rows.Next() {
		var req auth.PasswordResetRequest
		if err := rows.Scan(&req.ID, &req.UserID, &req.Email, &req.CodeHash, &req.ExpiresAt, &req.ConsumedAt,
			&req.RequestedByAdmin, &req.RequestedByUserID, &req.CreatedAt); err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}

// AccountMergeRequestRepo implements auth.AccountMergeRequestRepository.
type AccountMergeRequestRepo struct{ *DB }

func NewAccountMergeRequestRepo(db *DB) *AccountMergeRequestRepo {
	return &AccountMergeRequestRepo{DB: db}
}

func (r *AccountMergeRequestRepo) Save(ctx context.Context, req auth.AccountMergeRequest) (auth.AccountMergeRequest, error) {
	err := r.queryRow(ctx,
		`INSERT INTO account_merge_request (primary_user_id, secondary_user_id, status, verification_token, token_expires_at, completed_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, primary_user_id, secondary_user_id, status, verification_token, token_expires_at, completed_at, created_at`,
		req.PrimaryUserID, req.SecondaryUserID, req.Status, req.VerificationToken, req.TokenExpiresAt, req.CompletedAt, req.CreatedAt,
	).Scan(&req.ID, &req.PrimaryUserID, &req.SecondaryUserID, &req.Status, &req.VerificationToken,
		&req.TokenExpiresAt, &req.CompletedAt, &req.CreatedAt)
	if err != nil {
		return auth.AccountMergeRequest{}, err
	}
	return req, nil
}

func (r *AccountMergeRequestRepo) FindPendingBySecondaryUserID(ctx context.Context, secondaryUserID string) (*auth.AccountMergeRequest, error) {
	var req auth.AccountMergeRequest
	err := r.queryRow(ctx,
		`SELECT id, primary_user_id, secondary_user_id, status, verification_token, token_expires_at, completed_at, created_at
		 FROM account_merge_request WHERE secondary_user_id = $1 AND status = 'PENDING'`, secondaryUserID,
	).Scan(&req.ID, &req.PrimaryUserID, &req.SecondaryUserID, &req.Status, &req.VerificationToken,
		&req.TokenExpiresAt, &req.CompletedAt, &req.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}
