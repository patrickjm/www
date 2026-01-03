package app

import (
	"testing"

	"www/internal/daemon"
)

func TestResolveTabIDFromStatus(t *testing.T) {
	_, err := resolveTabIDFromStatus(daemon.StatusResult{})
	if err == nil {
		t.Fatalf("expected error for no tabs")
	}
	id, err := resolveTabIDFromStatus(daemon.StatusResult{Tabs: []daemon.TabInfo{{ID: 3}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 3 {
		t.Fatalf("expected tab 3, got %d", id)
	}
	_, err = resolveTabIDFromStatus(daemon.StatusResult{Tabs: []daemon.TabInfo{{ID: 1}, {ID: 2}}})
	if err == nil {
		t.Fatalf("expected error for multiple tabs")
	}
}
