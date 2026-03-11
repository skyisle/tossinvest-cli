package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

type LoginConfig struct {
	PythonBin        string
	HelperDir        string
	StorageStatePath string
}

type Options struct {
	LoginConfig LoginConfig
	Runner      LoginRunner
	Validator   SessionValidator
}

type Service struct {
	store       session.Store
	sessionFile string
	loginConfig LoginConfig
	runner      LoginRunner
	validator   SessionValidator
}

type Status struct {
	Active          bool       `json:"active"`
	Expired         bool       `json:"expired"`
	Provider        string     `json:"provider,omitempty"`
	CookieCount     int        `json:"cookie_count,omitempty"`
	StorageKeys     int        `json:"storage_keys,omitempty"`
	RetrievedAt     *time.Time `json:"retrieved_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	SessionFile     string     `json:"session_file"`
	Validated       bool       `json:"validated"`
	Valid           bool       `json:"valid"`
	ValidationError string     `json:"validation_error,omitempty"`
	CheckedAt       *time.Time `json:"checked_at,omitempty"`
}

type SessionValidator interface {
	ValidateSession(context.Context) error
}

func DefaultLoginConfig(cacheDir string) LoginConfig {
	pythonBin := os.Getenv("TOSSCTL_AUTH_HELPER_PYTHON")
	if pythonBin == "" {
		pythonBin = "python3"
	}

	helperDir := os.Getenv("TOSSCTL_AUTH_HELPER_DIR")
	if helperDir == "" {
		helperDir = "auth-helper"
	}

	storageStatePath := os.Getenv("TOSSCTL_AUTH_STORAGE_STATE")
	if storageStatePath == "" {
		storageStatePath = filepath.Join(cacheDir, "auth", "playwright-storage-state.json")
	}

	return LoginConfig{
		PythonBin:        pythonBin,
		HelperDir:        helperDir,
		StorageStatePath: storageStatePath,
	}
}

func NewService(store session.Store, sessionFile string, opts Options) *Service {
	runner := opts.Runner
	if runner == nil {
		runner = PythonLoginRunner{}
	}

	return &Service{
		store:       store,
		sessionFile: sessionFile,
		loginConfig: opts.LoginConfig,
		runner:      runner,
		validator:   opts.Validator,
	}
}

func (s *Service) Login(ctx context.Context) (*session.Session, error) {
	result, err := s.runner.Login(ctx, s.loginConfig)
	if err != nil {
		return nil, err
	}

	return s.ImportPlaywrightState(ctx, result.StorageStatePath)
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	sess, err := s.store.Load(ctx)
	if err != nil {
		if errors.Is(err, session.ErrNoSession) {
			return Status{
				Active:      false,
				Expired:     false,
				SessionFile: s.sessionFile,
			}, nil
		}

		return Status{}, err
	}

	status := Status{
		Active:      true,
		Expired:     sess.IsExpired(time.Now()),
		Provider:    sess.Provider,
		CookieCount: len(sess.Cookies),
		StorageKeys: len(sess.Storage),
		RetrievedAt: &sess.RetrievedAt,
		ExpiresAt:   sess.ExpiresAt,
		SessionFile: s.sessionFile,
	}

	if s.validator != nil {
		checkedAt := time.Now().UTC()
		status.Validated = true
		status.CheckedAt = &checkedAt
		if err := s.validator.ValidateSession(ctx); err != nil {
			status.Valid = false
			status.ValidationError = err.Error()
			return status, nil
		}
		status.Valid = true
	}

	return status, nil
}

func (s *Service) Logout(ctx context.Context) (bool, error) {
	if _, err := s.store.Load(ctx); err != nil {
		if errors.Is(err, session.ErrNoSession) {
			return false, nil
		}

		return false, err
	}

	if err := s.store.Clear(ctx); err != nil {
		return false, err
	}

	return true, nil
}
