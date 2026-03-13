package orderlineage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	CurrentOrderID string    `json:"current_order_id"`
	Kind           string    `json:"kind,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type File struct {
	Mappings map[string]Entry `json:"mappings"`
}

type Service struct {
	path string
}

func NewService(path string) *Service {
	return &Service{path: path}
}

func (s *Service) Path() string {
	return s.path
}

func (s *Service) Resolve(orderID string) (string, bool, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return "", false, nil
	}

	state, err := s.load()
	if err != nil {
		return "", false, err
	}

	current := orderID
	seen := map[string]struct{}{}
	for {
		if _, ok := seen[current]; ok {
			break
		}
		seen[current] = struct{}{}

		entry, ok := state.Mappings[current]
		if !ok {
			break
		}
		next := strings.TrimSpace(entry.CurrentOrderID)
		if next == "" || next == current {
			break
		}
		current = next
	}

	if current == orderID {
		return "", false, nil
	}
	return current, true, nil
}

func (s *Service) Record(originalOrderID, currentOrderID, kind string) error {
	originalOrderID = strings.TrimSpace(originalOrderID)
	currentOrderID = strings.TrimSpace(currentOrderID)
	kind = strings.TrimSpace(kind)
	if originalOrderID == "" || currentOrderID == "" || originalOrderID == currentOrderID {
		return nil
	}

	state, err := s.load()
	if err != nil {
		return err
	}
	if state.Mappings == nil {
		state.Mappings = map[string]Entry{}
	}

	now := time.Now().UTC()
	state.Mappings[originalOrderID] = Entry{
		CurrentOrderID: currentOrderID,
		Kind:           kind,
		UpdatedAt:      now,
	}
	for alias, entry := range state.Mappings {
		if strings.TrimSpace(entry.CurrentOrderID) != originalOrderID {
			continue
		}
		entry.CurrentOrderID = currentOrderID
		entry.UpdatedAt = now
		if kind != "" {
			entry.Kind = kind
		}
		state.Mappings[alias] = entry
	}

	return s.save(state)
}

func (s *Service) load() (File, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{Mappings: map[string]Entry{}}, nil
		}
		return File{}, err
	}
	if len(data) == 0 {
		return File{Mappings: map[string]Entry{}}, nil
	}

	var state File
	if err := json.Unmarshal(data, &state); err != nil {
		return File{}, err
	}
	if state.Mappings == nil {
		state.Mappings = map[string]Entry{}
	}
	return state, nil
}

func (s *Service) save(state File) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
