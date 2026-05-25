package persist

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// AdminStore 持久化管理面板的"临时隐藏用户"列表到 data/admin.json。
type AdminStore struct {
	mu    sync.RWMutex
	path  string
	state state
}

type state struct {
	HiddenUserIDs []int64 `json:"hidden_user_ids"`
}

func NewAdminStore(path string) (*AdminStore, error) {
	s := &AdminStore{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *AdminStore) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return json.Unmarshal(b, &s.state)
}

func (s *AdminStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *AdminStore) HiddenUserIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]int64, len(s.state.HiddenUserIDs))
	copy(out, s.state.HiddenUserIDs)
	return out
}

func (s *AdminStore) AddHidden(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.state.HiddenUserIDs {
		if x == id {
			return nil
		}
	}
	s.state.HiddenUserIDs = append(s.state.HiddenUserIDs, id)
	sort.Slice(s.state.HiddenUserIDs, func(i, j int) bool {
		return s.state.HiddenUserIDs[i] < s.state.HiddenUserIDs[j]
	})
	return s.save()
}

func (s *AdminStore) RemoveHidden(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.state.HiddenUserIDs[:0]
	for _, x := range s.state.HiddenUserIDs {
		if x != id {
			out = append(out, x)
		}
	}
	s.state.HiddenUserIDs = out
	return s.save()
}
