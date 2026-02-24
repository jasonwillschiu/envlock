package backend

import (
	"context"

	"github.com/jasonchiu/envlock/internal/enroll"
	"github.com/jasonchiu/envlock/internal/recipients"
)

// Store is the minimal remote metadata interface used by the current CLI.
// It is intentionally small so we can swap Tigris for a server-backed API
// implementation without rewriting command handlers all at once.
type Store interface {
	LoadRecipients(ctx context.Context) (recipients.Store, error)
	WriteRecipients(ctx context.Context, rs recipients.Store) error

	SaveInvite(ctx context.Context, invite enroll.Invite) error
	LoadInvite(ctx context.Context, id string) (enroll.Invite, error)
	ListInvites(ctx context.Context) ([]enroll.Invite, error)

	SaveRequest(ctx context.Context, req enroll.Request) error
	LoadRequest(ctx context.Context, id string) (enroll.Request, error)
	ListRequests(ctx context.Context) ([]enroll.Request, error)
}
