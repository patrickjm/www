package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

type Info struct {
	PID           int       `json:"pid"`
	Socket        string    `json:"socket"`
	StartedAt     time.Time `json:"started_at"`
	BinaryPath    string    `json:"binary_path,omitempty"`
	BinaryModTime time.Time `json:"binary_mod_time,omitempty"`
}

type Manager struct {
	ProfileDir string
	BinaryPath string
}

func (m Manager) SocketPath(profile string) string {
	return filepath.Join(m.ProfileDir, profile, "daemon.sock")
}

func (m Manager) InfoPath(profile string) string {
	return filepath.Join(m.ProfileDir, profile, "daemon.json")
}

func (m Manager) LoadInfo(profile string) (Info, error) {
	b, err := os.ReadFile(m.InfoPath(profile))
	if err != nil {
		return Info{}, err
	}
	var info Info
	if err := json.Unmarshal(b, &info); err != nil {
		return Info{}, err
	}
	return info, nil
}

func (m Manager) SaveInfo(profile string, info Info) error {
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.InfoPath(profile), b, 0o644)
}

func (m Manager) IsRunning(profile string) (bool, Info, error) {
	info, err := m.LoadInfo(profile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, Info{}, nil
		}
		return false, Info{}, err
	}
	if !processAlive(info.PID) {
		_ = m.cleanupStale(profile)
		return false, Info{}, nil
	}
	if !socketAlive(info.Socket) {
		_ = m.cleanupStale(profile)
		return false, Info{}, nil
	}
	if m.binaryMismatch(info) {
		_ = m.Stop(profile)
		_ = m.cleanupStale(profile)
		return false, Info{}, nil
	}
	return true, info, nil
}

func (m Manager) Start(profile string) error {
	running, _, err := m.IsRunning(profile)
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	if m.BinaryPath == "" {
		path, err := os.Executable()
		if err != nil {
			return err
		}
		m.BinaryPath = path
	}
	cmd := exec.Command(m.BinaryPath, "--profile", profile, "--profile-dir", m.ProfileDir, "serve")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if socketAlive(m.SocketPath(profile)) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("daemon did not start")
}

func (m Manager) Stop(profile string) error {
	client, err := NewClient(m.SocketPath(profile))
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Stop()
}

func (m Manager) cleanupStale(profile string) error {
	_ = os.Remove(m.SocketPath(profile))
	_ = os.Remove(m.InfoPath(profile))
	return nil
}

func (m Manager) binaryMismatch(info Info) bool {
	path, modTime, err := CurrentBinaryInfo()
	if err != nil {
		return false
	}
	if info.BinaryPath == "" || info.BinaryModTime.IsZero() {
		return false
	}
	if info.BinaryPath != path {
		return true
	}
	return !info.BinaryModTime.Equal(modTime)
}

func socketAlive(path string) bool {
	conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func CurrentBinaryInfo() (string, time.Time, error) {
	path, err := os.Executable()
	if err != nil {
		return "", time.Time{}, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return path, time.Time{}, err
	}
	return path, stat.ModTime().UTC(), nil
}

func (m Manager) RunningProfiles() ([]Info, error) {
	entries, err := os.ReadDir(m.ProfileDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	infos := []Info{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		running, info, err := m.IsRunning(name)
		if err != nil {
			return nil, err
		}
		if running {
			infos = append(infos, info)
		}
	}
	return infos, nil
}

func EnsureProfileDir(path string) error {
	if path == "" {
		return errors.New("profile dir required")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}
	return nil
}
