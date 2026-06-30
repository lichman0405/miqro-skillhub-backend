package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/namespace"
)

// NamespaceRepo implements namespace.NamespaceRepository.
type NamespaceRepo struct {
	DB *DB
}

func NewNamespaceRepo(db *DB) *NamespaceRepo {
	return &NamespaceRepo{DB: db}
}

func (r *NamespaceRepo) FindByID(ctx context.Context, id int64) (*namespace.Namespace, error) {
	var n namespace.Namespace
	err := r.DB.queryRow(ctx,
		`SELECT id, slug, display_name, type, description, avatar_url, status, created_by, created_at, updated_at
		 FROM namespace WHERE id = $1`, id,
	).Scan(&n.ID, &n.Slug, &n.DisplayName, &n.Type, &n.Description, &n.AvatarURL, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NamespaceRepo) FindByIDs(ctx context.Context, ids []int64) ([]namespace.Namespace, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, slug, display_name, type, description, avatar_url, status, created_by, created_at, updated_at
		 FROM namespace WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nss []namespace.Namespace
	for rows.Next() {
		var n namespace.Namespace
		if err := rows.Scan(&n.ID, &n.Slug, &n.DisplayName, &n.Type, &n.Description, &n.AvatarURL, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		nss = append(nss, n)
	}
	return nss, rows.Err()
}

func (r *NamespaceRepo) FindBySlug(ctx context.Context, slug string) (*namespace.Namespace, error) {
	var n namespace.Namespace
	err := r.DB.queryRow(ctx,
		`SELECT id, slug, display_name, type, description, avatar_url, status, created_by, created_at, updated_at
		 FROM namespace WHERE slug = $1`, slug,
	).Scan(&n.ID, &n.Slug, &n.DisplayName, &n.Type, &n.Description, &n.AvatarURL, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NamespaceRepo) FindByStatus(ctx context.Context, status string) ([]namespace.Namespace, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, slug, display_name, type, description, avatar_url, status, created_by, created_at, updated_at
		 FROM namespace WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nss []namespace.Namespace
	for rows.Next() {
		var n namespace.Namespace
		if err := rows.Scan(&n.ID, &n.Slug, &n.DisplayName, &n.Type, &n.Description, &n.AvatarURL, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		nss = append(nss, n)
	}
	return nss, rows.Err()
}

func (r *NamespaceRepo) Save(ctx context.Context, n namespace.Namespace) (namespace.Namespace, error) {
	now := time.Now()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}
	n.UpdatedAt = now

	err := r.DB.queryRow(ctx,
		`INSERT INTO namespace (slug, display_name, type, description, avatar_url, status, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (slug) DO UPDATE SET
		   display_name = EXCLUDED.display_name,
		   type = EXCLUDED.type,
		   description = EXCLUDED.description,
		   avatar_url = EXCLUDED.avatar_url,
		   status = EXCLUDED.status,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, slug, display_name, type, description, avatar_url, status, created_by, created_at, updated_at`,
		n.Slug, n.DisplayName, n.Type, n.Description, n.AvatarURL, n.Status, n.CreatedBy, n.CreatedAt, n.UpdatedAt,
	).Scan(&n.ID, &n.Slug, &n.DisplayName, &n.Type, &n.Description, &n.AvatarURL, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return namespace.Namespace{}, err
	}
	return n, nil
}

func (r *NamespaceRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM namespace WHERE id = $1`, id)
	return err
}

// NamespaceMemberRepo implements namespace.NamespaceMemberRepository.
type NamespaceMemberRepo struct {
	DB *DB
}

func NewNamespaceMemberRepo(db *DB) *NamespaceMemberRepo {
	return &NamespaceMemberRepo{DB: db}
}

func (r *NamespaceMemberRepo) Save(ctx context.Context, m namespace.NamespaceMember) (namespace.NamespaceMember, error) {
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	err := r.DB.queryRow(ctx,
		`INSERT INTO namespace_member (namespace_id, user_id, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (namespace_id, user_id) DO UPDATE SET
		   role = EXCLUDED.role,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, namespace_id, user_id, role, created_at, updated_at`,
		m.NamespaceID, m.UserID, m.Role, m.CreatedAt, m.UpdatedAt,
	).Scan(&m.ID, &m.NamespaceID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return namespace.NamespaceMember{}, err
	}
	return m, nil
}

func (r *NamespaceMemberRepo) FindByNamespaceAndUser(ctx context.Context, namespaceID int64, userID string) (*namespace.NamespaceMember, error) {
	var m namespace.NamespaceMember
	err := r.DB.queryRow(ctx,
		`SELECT id, namespace_id, user_id, role, created_at, updated_at
		 FROM namespace_member WHERE namespace_id = $1 AND user_id = $2`, namespaceID, userID,
	).Scan(&m.ID, &m.NamespaceID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *NamespaceMemberRepo) FindByUserID(ctx context.Context, userID string) ([]namespace.NamespaceMember, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, namespace_id, user_id, role, created_at, updated_at
		 FROM namespace_member WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []namespace.NamespaceMember
	for rows.Next() {
		var m namespace.NamespaceMember
		if err := rows.Scan(&m.ID, &m.NamespaceID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *NamespaceMemberRepo) FindByNamespaceID(ctx context.Context, namespaceID int64) ([]namespace.NamespaceMember, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, namespace_id, user_id, role, created_at, updated_at
		 FROM namespace_member WHERE namespace_id = $1`, namespaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []namespace.NamespaceMember
	for rows.Next() {
		var m namespace.NamespaceMember
		if err := rows.Scan(&m.ID, &m.NamespaceID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *NamespaceMemberRepo) FindByNamespaceIDAndRoles(ctx context.Context, namespaceID int64, roles []string) ([]namespace.NamespaceMember, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, namespace_id, user_id, role, created_at, updated_at
		 FROM namespace_member WHERE namespace_id = $1 AND role = ANY($2)`, namespaceID, roles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []namespace.NamespaceMember
	for rows.Next() {
		var m namespace.NamespaceMember
		if err := rows.Scan(&m.ID, &m.NamespaceID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *NamespaceMemberRepo) DeleteByNamespaceAndUser(ctx context.Context, namespaceID int64, userID string) error {
	_, err := r.DB.exec(ctx,
		`DELETE FROM namespace_member WHERE namespace_id = $1 AND user_id = $2`, namespaceID, userID)
	return err
}

func (r *NamespaceMemberRepo) DeleteByNamespaceID(ctx context.Context, namespaceID int64) error {
	_, err := r.DB.exec(ctx,
		`DELETE FROM namespace_member WHERE namespace_id = $1`, namespaceID)
	return err
}
