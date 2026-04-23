package permissions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrNoGrant      = errors.New("no active trading permission grant")
	ErrExpiredGrant = errors.New("trading permission grant expired")
)

type Grant struct {
	GrantedAt time.Time `json:"granted_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Status struct {
	Active         bool       `json:"active"`
	Expired        bool       `json:"expired"`
	GrantedAt      *time.Time `json:"granted_at,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Remaining      int64      `json:"remaining_seconds,omitempty"`
	PermissionFile string     `json:"permission_file"`
}

type Service struct {
	path string
	now  func() time.Time
}

func NewService(path string) *Service {
	return &Service{
		path: path,
		now:  time.Now,
	}
}

func (s *Service) Grant(ctx context.Context, ttl time.Duration) (Status, error) {
	now := s.now().UTC()
	grant := Grant{
		GrantedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	if err := s.save(ctx, grant); err != nil {
		return Status{}, err
	}

	return s.Status(ctx)
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	grant, err := s.load(ctx)
	if err != nil {
		if errors.Is(err, ErrNoGrant) {
			return Status{PermissionFile: s.path}, nil
		}
		return Status{}, err
	}

	now := s.now().UTC()
	expired := now.After(grant.ExpiresAt)
	status := Status{
		Active:         !expired,
		Expired:        expired,
		GrantedAt:      &grant.GrantedAt,
		ExpiresAt:      &grant.ExpiresAt,
		PermissionFile: s.path,
	}
	if !expired {
		status.Remaining = int64(grant.ExpiresAt.Sub(now).Seconds())
	}
	return status, nil
}

func (s *Service) Revoke(context.Context) (bool, error) {
	if err := os.Remove(s.path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) Require(ctx context.Context) error {
	grant, err := s.load(ctx)
	if err != nil {
		return err
	}

	if s.now().UTC().After(grant.ExpiresAt) {
		return ErrExpiredGrant
	}

	return nil
}

func (s *Service) load(context.Context) (Grant, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Grant{}, ErrNoGrant
		}
		return Grant{}, err
	}

	var grant Grant
	if err := json.Unmarshal(data, &grant); err != nil {
		return Grant{}, err
	}

	return grant, nil
}

func (s *Service) save(_ context.Context, grant Grant) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(grant, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}
