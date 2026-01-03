package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ProfileDir string
	DefaultTTL time.Duration
}

type rawConfig struct {
	ProfileDir string `toml:"profile_dir"`
	DefaultTTL string `toml:"default_ttl"`
}

func Load(profileDirOverride string, defaultTTLOverride string) (Config, error) {
	cfg := Config{
		ProfileDir: defaultProfileDir(),
		DefaultTTL: 14 * 24 * time.Hour,
	}

	if err := loadSystemConfig(&cfg); err != nil {
		return Config{}, err
	}

	if v := strings.TrimSpace(os.Getenv("WWW_PROFILE_DIR")); v != "" {
		cfg.ProfileDir = v
	}
	if v := strings.TrimSpace(os.Getenv("WWW_DEFAULT_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.DefaultTTL = d
		}
	}
	if strings.TrimSpace(defaultTTLOverride) != "" {
		if d, err := time.ParseDuration(defaultTTLOverride); err == nil {
			cfg.DefaultTTL = d
		}
	}
	if strings.TrimSpace(profileDirOverride) != "" {
		cfg.ProfileDir = profileDirOverride
	}

	return cfg, nil
}

func loadSystemConfig(cfg *Config) error {
	paths := []string{
		"/opt/homebrew/etc/www/config.toml",
		"/usr/local/etc/www/config.toml",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		var raw rawConfig
		if _, err := toml.DecodeFile(path, &raw); err != nil {
			return err
		}
		if raw.ProfileDir != "" {
			cfg.ProfileDir = raw.ProfileDir
		}
		if raw.DefaultTTL != "" {
			if d, err := time.ParseDuration(raw.DefaultTTL); err == nil {
				cfg.DefaultTTL = d
			}
		}
		return nil
	}
	return nil
}

func defaultProfileDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/www"
	}
	userDefault := ""
	if runtime.GOOS == "darwin" {
		userDefault = filepath.Join(home, "Library", "Application Support", "www")
	} else {
		if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
			userDefault = filepath.Join(xdg, "www")
		} else {
			userDefault = filepath.Join(home, ".local", "share", "www")
		}
	}
	if isWritableDir("/opt/homebrew/var") {
		return "/opt/homebrew/var/www"
	}
	if isWritableDir("/usr/local/var") {
		return "/usr/local/var/www"
	}
	return userDefault
}

func isWritableDir(path string) bool {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return false
	}
	testFile := filepath.Join(path, ".www-writetest")
	if err := os.WriteFile(testFile, []byte("ok"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(testFile)
	return true
}
