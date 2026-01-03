package profile

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreUpsertLoad(t *testing.T) {
	dir := t.TempDir()
	store := Store{Root: dir, DefaultTTL: time.Hour}
	p, created, err := store.Upsert("Alice", Overrides{})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !created {
		t.Fatalf("expected created")
	}
	if p.Name != "alice" {
		t.Fatalf("expected sanitized name, got %s", p.Name)
	}
	loaded, err := store.Load("alice")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Name != "alice" {
		t.Fatalf("load name: %s", loaded.Name)
	}
	if _, err := store.Load("missing"); err == nil {
		t.Fatalf("expected error for missing profile")
	}
	if path := store.ProfilePath("alice"); filepath.Base(path) != "profile.json" {
		t.Fatalf("unexpected profile path: %s", path)
	}
}

func TestStoreExpiry(t *testing.T) {
	dir := t.TempDir()
	store := Store{Root: dir, DefaultTTL: time.Second}
	p, _, err := store.Upsert("expiring", Overrides{})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	p.LastUsed = time.Now().Add(-2 * time.Second)
	if err := store.Save(p); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := store.Load("expiring")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !store.IsExpired(loaded) {
		t.Fatalf("expected expired")
	}
	removed, err := store.Prune()
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(removed))
	}
}
