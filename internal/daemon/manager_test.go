package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBinaryMismatch(t *testing.T) {
	path, modTime, err := CurrentBinaryInfo()
	if err != nil {
		t.Fatalf("current binary info: %v", err)
	}
	info := Info{BinaryPath: path, BinaryModTime: modTime}
	mgr := Manager{}
	if mgr.binaryMismatch(info) {
		t.Fatalf("expected no mismatch for current binary")
	}
	info.BinaryModTime = modTime.Add(-time.Minute)
	if !mgr.binaryMismatch(info) {
		t.Fatalf("expected mismatch for mod time")
	}
	info.BinaryPath = filepath.Join(os.TempDir(), "nonexistent-binary")
	info.BinaryModTime = modTime
	if !mgr.binaryMismatch(info) {
		t.Fatalf("expected mismatch for path")
	}
}
