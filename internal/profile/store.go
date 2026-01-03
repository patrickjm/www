package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Profile struct {
	Name      string    `json:"name"`
	Browser   string    `json:"browser"`
	Channel   string    `json:"channel"`
	Headless  bool      `json:"headless"`
	TTL       int64     `json:"ttl_seconds"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

type Store struct {
	Root       string
	DefaultTTL time.Duration
}

func (s Store) EnsureDir() error {
	return os.MkdirAll(s.Root, 0o755)
}

func (s Store) ProfileDir(name string) string {
	return filepath.Join(s.Root, sanitizeName(name))
}

func (s Store) ProfilePath(name string) string {
	return filepath.Join(s.ProfileDir(name), "profile.json")
}

func (s Store) StorageStatePath(name string) string {
	return filepath.Join(s.ProfileDir(name), "storage.json")
}

func (s Store) Load(name string) (Profile, error) {
	path := s.ProfilePath(name)
	b, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s Store) Save(p Profile) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}
	p.Name = sanitizeName(p.Name)
	if p.Name == "" {
		return errors.New("profile name required")
	}
	if err := os.MkdirAll(s.ProfileDir(p.Name), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.ProfilePath(p.Name), b, 0o644)
}

func (s Store) List() ([]Profile, error) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	profiles := make([]Profile, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		p, err := s.Load(name)
		if err != nil {
			continue
		}
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}

func (s Store) Remove(name string) error {
	if name == "" {
		return errors.New("profile name required")
	}
	return os.RemoveAll(s.ProfileDir(name))
}

func (s Store) Upsert(name string, overrides Overrides) (Profile, bool, error) {
	name = sanitizeName(name)
	if name == "" {
		return Profile{}, false, errors.New("profile name required")
	}
	p, err := s.Load(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return Profile{}, false, err
		}
		p = Profile{
			Name:      name,
			Browser:   "chromium",
			Channel:   "chrome",
			Headless:  true,
			TTL:       int64(s.DefaultTTL.Seconds()),
			CreatedAt: time.Now().UTC(),
			LastUsed:  time.Now().UTC(),
		}
		applyOverrides(&p, overrides)
		if err := s.Save(p); err != nil {
			return Profile{}, false, err
		}
		return p, true, nil
	}
	updated := applyOverrides(&p, overrides)
	if updated {
		if err := s.Save(p); err != nil {
			return Profile{}, false, err
		}
	}
	return p, false, nil
}

func (s Store) Touch(name string) (Profile, error) {
	p, err := s.Load(name)
	if err != nil {
		return Profile{}, err
	}
	p.LastUsed = time.Now().UTC()
	return p, s.Save(p)
}

func (s Store) IsExpired(p Profile) bool {
	if p.TTL <= 0 {
		return false
	}
	deadline := p.LastUsed.Add(time.Duration(p.TTL) * time.Second)
	return time.Now().UTC().After(deadline)
}

func (s Store) Prune() ([]Profile, error) {
	profiles, err := s.List()
	if err != nil {
		return nil, err
	}
	removed := make([]Profile, 0)
	for _, p := range profiles {
		if s.IsExpired(p) {
			if err := s.Remove(p.Name); err != nil {
				return removed, err
			}
			removed = append(removed, p)
		}
	}
	return removed, nil
}

type Overrides struct {
	Browser  string
	Channel  string
	Headless *bool
	TTL      *time.Duration
}

func applyOverrides(p *Profile, overrides Overrides) bool {
	updated := false
	if overrides.Browser != "" {
		p.Browser = overrides.Browser
		updated = true
	}
	if overrides.Channel != "" {
		p.Channel = overrides.Channel
		updated = true
	}
	if overrides.Headless != nil {
		p.Headless = *overrides.Headless
		updated = true
	}
	if overrides.TTL != nil {
		p.TTL = int64(overrides.TTL.Seconds())
		updated = true
	}
	return updated
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

func SafeName(name string) string {
	return sanitizeName(name)
}

func FormatTTL(seconds int64) string {
	if seconds <= 0 {
		return "never"
	}
	return time.Duration(seconds * int64(time.Second)).String()
}

func (p Profile) String() string {
	return fmt.Sprintf("%s (browser=%s channel=%s headless=%t)", p.Name, p.Browser, p.Channel, p.Headless)
}
