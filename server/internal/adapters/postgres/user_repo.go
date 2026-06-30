package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/auth"
)

// UserAccountRepo implements auth.UserAccountRepository.
type UserAccountRepo struct {
	DB *DB
}

func NewUserAccountRepo(db *DB) *UserAccountRepo {
	return &UserAccountRepo{DB: db}
}

func (r *UserAccountRepo) FindByID(ctx context.Context, id string) (*auth.UserAccount, error) {
	var u auth.UserAccount
	err := r.DB.queryRow(ctx,
		`SELECT id, display_name, email, avatar_url, status, merged_to_user_id, system_account, created_at, updated_at
		 FROM user_account WHERE id = $1`, id,
	).Scan(&u.ID, &u.DisplayName, &u.Email, &u.AvatarURL, &u.Status, &u.MergedToUserID, &u.SystemAccount, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserAccountRepo) FindByIDs(ctx context.Context, ids []string) ([]auth.UserAccount, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, display_name, email, avatar_url, status, merged_to_user_id, system_account, created_at, updated_at
		 FROM user_account WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []auth.UserAccount
	for rows.Next() {
		var u auth.UserAccount
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.Email, &u.AvatarURL, &u.Status, &u.MergedToUserID, &u.SystemAccount, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserAccountRepo) FindByEmail(ctx context.Context, email string) (*auth.UserAccount, error) {
	var u auth.UserAccount
	err := r.DB.queryRow(ctx,
		`SELECT id, display_name, email, avatar_url, status, merged_to_user_id, system_account, created_at, updated_at
		 FROM user_account WHERE LOWER(email) = LOWER($1)`, email,
	).Scan(&u.ID, &u.DisplayName, &u.Email, &u.AvatarURL, &u.Status, &u.MergedToUserID, &u.SystemAccount, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserAccountRepo) Save(ctx context.Context, u auth.UserAccount) (auth.UserAccount, error) {
	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now

	_, err := r.DB.exec(ctx,
		`INSERT INTO user_account (id, display_name, email, avatar_url, status, merged_to_user_id, system_account, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO UPDATE SET
		   display_name = EXCLUDED.display_name,
		   email = EXCLUDED.email,
		   avatar_url = EXCLUDED.avatar_url,
		   status = EXCLUDED.status,
		   merged_to_user_id = EXCLUDED.merged_to_user_id,
		   system_account = EXCLUDED.system_account,
		   updated_at = EXCLUDED.updated_at`,
		u.ID, u.DisplayName, u.Email, u.AvatarURL, u.Status, u.MergedToUserID, u.SystemAccount, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return auth.UserAccount{}, err
	}
	return u, nil
}
