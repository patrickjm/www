package daemon

import (
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/patrickjm/www/internal/browser"
)

func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", path, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return errors.New("socket not ready")
}

func TestServerTabLifecycle(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "daemon.sock")
	engine := &browser.FakeEngine{}
	opts := browser.StartOptions{Headless: true}

	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeProfile(socket, "test", engine, opts)
	}()
	if err := waitForSocket(socket, 2*time.Second); err != nil {
		t.Fatalf("wait socket: %v", err)
	}
	client, err := NewClient(socket)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer client.Close()

	tabs, err := client.TabList()
	if err != nil {
		t.Fatalf("tab list: %v", err)
	}
	if len(tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(tabs))
	}
	url, err := client.URL(tabs[0].ID)
	if err != nil {
		t.Fatalf("url: %v", err)
	}
	if url != "" {
		t.Fatalf("expected empty url, got %s", url)
	}
	links, err := client.Links(tabs[0].ID, "")
	if err != nil {
		t.Fatalf("links: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("expected no links, got %d", len(links))
	}
	newTab, err := client.TabNew("")
	if err != nil {
		t.Fatalf("tab new: %v", err)
	}
	if newTab.ID == 0 {
		t.Fatalf("expected tab id")
	}
	if err := client.Goto(newTab.ID, "https://example.com", 1000); err != nil {
		t.Fatalf("goto: %v", err)
	}
	if err := client.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server error: %v", err)
	}
	if engine.Session == nil || len(engine.Session.Pages) < 2 {
		t.Fatalf("expected pages to be created")
	}
}
