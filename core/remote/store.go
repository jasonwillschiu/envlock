package remote

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"

	"github.com/jasonchiu/envlock/core/config"
	"github.com/jasonchiu/envlock/core/tigris"
	"github.com/jasonchiu/envlock/feature/enroll"
	"github.com/jasonchiu/envlock/feature/recipients"
)

type Store struct {
	client *tigris.Client
	prefix string
}

func New(ctx context.Context, proj config.Project) (*Store, error) {
	client, err := tigris.NewFromProject(proj)
	if err != nil {
		return nil, err
	}
	pfx := strings.Trim(strings.TrimSpace(proj.Prefix), "/")
	if pfx == "" {
		pfx = config.DefaultPrefix(proj.AppName)
	}
	if pfx == "" {
		return nil, errors.New("project prefix is required")
	}
	return &Store{client: client, prefix: pfx}, nil
}

func (s *Store) recipientsKey() string {
	return path.Join(s.prefix, "_envlock", "recipients.json")
}

func (s *Store) inviteKey(id string) string {
	return path.Join(s.prefix, "_envlock", "enroll", "invites", strings.TrimSpace(id)+".json")
}

func (s *Store) requestKey(id string) string {
	return path.Join(s.prefix, "_envlock", "enroll", "requests", strings.TrimSpace(id)+".json")
}

func (s *Store) invitesPrefix() string {
	return path.Join(s.prefix, "_envlock", "enroll", "invites") + "/"
}

func (s *Store) requestsPrefix() string {
	return path.Join(s.prefix, "_envlock", "enroll", "requests") + "/"
}

func (s *Store) LoadRecipients(ctx context.Context) (recipients.Store, error) {
	var rs recipients.Store
	err := s.client.GetJSON(ctx, s.recipientsKey(), &rs)
	if err != nil {
		if errors.Is(err, tigris.ErrObjectNotFound) {
			return recipients.Store{Version: 1, Recipients: []recipients.Recipient{}}, nil
		}
		return recipients.Store{}, err
	}
	if rs.Version == 0 {
		rs.Version = 1
	}
	return rs, nil
}

func (s *Store) WriteRecipients(ctx context.Context, rs recipients.Store) error {
	if rs.Version == 0 {
		rs.Version = 1
	}
	return s.client.PutJSON(ctx, s.recipientsKey(), rs)
}

func (s *Store) SaveInvite(ctx context.Context, invite enroll.Invite) error {
	return s.client.PutJSON(ctx, s.inviteKey(invite.ID), invite)
}

func (s *Store) LoadInvite(ctx context.Context, id string) (enroll.Invite, error) {
	var inv enroll.Invite
	err := s.client.GetJSON(ctx, s.inviteKey(id), &inv)
	if err != nil {
		if errors.Is(err, tigris.ErrObjectNotFound) {
			return enroll.Invite{}, enroll.ErrInviteNotFound
		}
		return enroll.Invite{}, err
	}
	if inv.Version == 0 {
		inv.Version = 1
	}
	return inv, nil
}

func (s *Store) ListInvites(ctx context.Context) ([]enroll.Invite, error) {
	keys, err := s.client.ListKeys(ctx, s.invitesPrefix())
	if err != nil {
		return nil, err
	}
	out := make([]enroll.Invite, 0, len(keys))
	for _, key := range keys {
		var inv enroll.Invite
		if err := s.client.GetJSON(ctx, key, &inv); err != nil {
			return nil, err
		}
		if inv.Version == 0 {
			inv.Version = 1
		}
		out = append(out, inv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) SaveRequest(ctx context.Context, req enroll.Request) error {
	return s.client.PutJSON(ctx, s.requestKey(req.ID), req)
}

func (s *Store) LoadRequest(ctx context.Context, id string) (enroll.Request, error) {
	var req enroll.Request
	err := s.client.GetJSON(ctx, s.requestKey(id), &req)
	if err != nil {
		if errors.Is(err, tigris.ErrObjectNotFound) {
			return enroll.Request{}, enroll.ErrRequestNotFound
		}
		return enroll.Request{}, err
	}
	if req.Version == 0 {
		req.Version = 1
	}
	return req, nil
}

func (s *Store) ListRequests(ctx context.Context) ([]enroll.Request, error) {
	keys, err := s.client.ListKeys(ctx, s.requestsPrefix())
	if err != nil {
		return nil, err
	}
	out := make([]enroll.Request, 0, len(keys))
	for _, key := range keys {
		var req enroll.Request
		if err := s.client.GetJSON(ctx, key, &req); err != nil {
			return nil, err
		}
		if req.Version == 0 {
			req.Version = 1
		}
		out = append(out, req)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
