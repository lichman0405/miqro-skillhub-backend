package namespace

import "context"

// NamespaceRepository defines the persistence contract for namespaces.
type NamespaceRepository interface {
	FindByID(ctx context.Context, id int64) (*Namespace, error)
	FindByIDs(ctx context.Context, ids []int64) ([]Namespace, error)
	FindBySlug(ctx context.Context, slug string) (*Namespace, error)
	FindByStatus(ctx context.Context, status string) ([]Namespace, error)
	Save(ctx context.Context, ns Namespace) (Namespace, error)
	Delete(ctx context.Context, id int64) error
}

// NamespaceMemberRepository defines the persistence contract for namespace members.
type NamespaceMemberRepository interface {
	Save(ctx context.Context, member NamespaceMember) (NamespaceMember, error)
	FindByNamespaceAndUser(ctx context.Context, namespaceID int64, userID string) (*NamespaceMember, error)
	FindByUserID(ctx context.Context, userID string) ([]NamespaceMember, error)
	FindByNamespaceID(ctx context.Context, namespaceID int64) ([]NamespaceMember, error)
	FindByNamespaceIDAndRoles(ctx context.Context, namespaceID int64, roles []string) ([]NamespaceMember, error)
	DeleteByNamespaceAndUser(ctx context.Context, namespaceID int64, userID string) error
	DeleteByNamespaceID(ctx context.Context, namespaceID int64) error
}
