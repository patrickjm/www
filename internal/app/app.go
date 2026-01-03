package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/patrickjm/www/internal/browser"
	"github.com/patrickjm/www/internal/config"
	"github.com/patrickjm/www/internal/daemon"
	"github.com/patrickjm/www/internal/profile"
)

type GlobalFlags struct {
	Profile    string
	ProfileDir string
	JSON       bool
	Plain      bool
	Quiet      bool
	Verbose    bool
	NoStart    bool
	Save       bool
	Browser    string
	Channel    string
	Headless   bool
	Headed     bool
	Tab        int
	TTL        string
	Selector   string
	Main       bool
	Timeout    string
}

type App struct {
	Out io.Writer
	Err io.Writer
}

func (a App) prepare(flags GlobalFlags) (config.Config, profile.Store, daemon.Manager, error) {
	cfg, err := config.Load(flags.ProfileDir, "")
	if err != nil {
		return config.Config{}, profile.Store{}, daemon.Manager{}, err
	}
	store := profile.Store{Root: cfg.ProfileDir, DefaultTTL: cfg.DefaultTTL}
	if err := daemon.EnsureProfileDir(cfg.ProfileDir); err != nil {
		return config.Config{}, profile.Store{}, daemon.Manager{}, err
	}
	mgr := daemon.Manager{ProfileDir: cfg.ProfileDir}
	return cfg, store, mgr, nil
}

const (
	exitSuccess  = 0
	exitFailure  = 1
	exitUsage    = 2
	exitNotFound = 3
)

func (a App) runInstall(flags GlobalFlags) int {
	browsers := []string{}
	if flags.Browser != "" {
		browsers = append(browsers, flags.Browser)
	}
	opts := &playwright.RunOptions{}
	if len(browsers) > 0 {
		opts.Browsers = browsers
	}
	if err := playwright.Install(opts); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if !flags.Quiet {
		if len(browsers) == 0 {
			fmt.Fprintln(a.Out, "Playwright installed")
		} else {
			fmt.Fprintf(a.Out, "Playwright installed: %s\n", strings.Join(browsers, ", "))
		}
	}
	return exitSuccess
}

func (a App) runDoctor(cfg config.Config, flags GlobalFlags) int {
	type result struct {
		ProfileDirWritable bool   `json:"profile_dir_writable"`
		ProfileDir         string `json:"profile_dir"`
		PlaywrightOK       bool   `json:"playwright_ok"`
		BrowsersPath       string `json:"browsers_path"`
	}
	res := result{ProfileDir: cfg.ProfileDir, BrowsersPath: os.Getenv("PLAYWRIGHT_BROWSERS_PATH")}
	if err := os.MkdirAll(cfg.ProfileDir, 0o755); err == nil {
		res.ProfileDirWritable = true
	}
	if pw, err := playwright.Run(); err == nil {
		res.PlaywrightOK = true
		pw.Stop()
	}
	if flags.JSON {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return exitSuccess
	}
	fmt.Fprintf(a.Out, "profile_dir=%s\n", res.ProfileDir)
	fmt.Fprintf(a.Out, "profile_dir_writable=%t\n", res.ProfileDirWritable)
	fmt.Fprintf(a.Out, "playwright_ok=%t\n", res.PlaywrightOK)
	if res.BrowsersPath != "" {
		fmt.Fprintf(a.Out, "browsers_path=%s\n", res.BrowsersPath)
	}
	return exitSuccess
}

func (a App) runStart(store profile.Store, mgr daemon.Manager, flags GlobalFlags) int {
	name := flags.Profile
	if name == "" {
		fmt.Fprintln(a.Err, "-p/--profile is required")
		return exitUsage
	}
	overrides, err := overridesFromFlags(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	p, _, err := store.Upsert(name, overrides)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if err := mgr.Start(p.Name); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	_, _ = store.Touch(p.Name)
	if !flags.Quiet {
		fmt.Fprintf(a.Out, "started %s\n", p.Name)
	}
	return exitSuccess
}

func (a App) runStop(mgr daemon.Manager, flags GlobalFlags) int {
	name := flags.Profile
	if name == "" {
		fmt.Fprintln(a.Err, "-p/--profile is required")
		return exitUsage
	}
	if err := mgr.Stop(profile.SafeName(name)); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if !flags.Quiet {
		fmt.Fprintf(a.Out, "stopped %s\n", name)
	}
	return exitSuccess
}

func (a App) runPs(mgr daemon.Manager, flags GlobalFlags) int {
	infos, err := mgr.RunningProfiles()
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if flags.JSON {
		b, _ := json.MarshalIndent(infos, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return exitSuccess
	}
	for _, info := range infos {
		fmt.Fprintf(a.Out, "pid=%d socket=%s started_at=%s\n", info.PID, info.Socket, info.StartedAt.Format(time.RFC3339))
	}
	return exitSuccess
}

func (a App) runList(store profile.Store, flags GlobalFlags) int {
	profiles, err := store.List()
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if flags.JSON {
		b, _ := json.MarshalIndent(profiles, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return exitSuccess
	}
	for _, p := range profiles {
		fmt.Fprintf(a.Out, "%s last_used=%s ttl=%s\n", p.Name, p.LastUsed.Format(time.RFC3339), profile.FormatTTL(p.TTL))
	}
	return exitSuccess
}

func (a App) runShow(store profile.Store, flags GlobalFlags, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(a.Err, "profile name required")
		return exitUsage
	}
	p, err := store.Load(args[0])
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitNotFound
	}
	if flags.JSON {
		b, _ := json.MarshalIndent(p, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return exitSuccess
	}
	fmt.Fprintf(a.Out, "name=%s\n", p.Name)
	fmt.Fprintf(a.Out, "browser=%s channel=%s\n", p.Browser, p.Channel)
	fmt.Fprintf(a.Out, "headless=%t\n", p.Headless)
	fmt.Fprintf(a.Out, "created_at=%s\n", p.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(a.Out, "last_used=%s\n", p.LastUsed.Format(time.RFC3339))
	fmt.Fprintf(a.Out, "ttl=%s\n", profile.FormatTTL(p.TTL))
	return exitSuccess
}

func (a App) runRemove(store profile.Store, mgr daemon.Manager, flags GlobalFlags, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.Err, "profile name required")
		return exitUsage
	}
	for _, name := range args {
		running, _, err := mgr.IsRunning(name)
		if err != nil {
			fmt.Fprintln(a.Err, err)
			return exitFailure
		}
		if running {
			fmt.Fprintf(a.Err, "%s is running; stop first\n", name)
			return exitFailure
		}
		if err := store.Remove(name); err != nil {
			fmt.Fprintln(a.Err, err)
			return exitFailure
		}
		if !flags.Quiet {
			fmt.Fprintf(a.Out, "removed %s\n", name)
		}
	}
	return exitSuccess
}

func (a App) runPrune(store profile.Store, mgr daemon.Manager, flags GlobalFlags, dryRun bool, force bool) int {
	profiles, err := store.List()
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	removed := []profile.Profile{}
	for _, p := range profiles {
		if !store.IsExpired(p) {
			continue
		}
		running, _, err := mgr.IsRunning(p.Name)
		if err != nil {
			fmt.Fprintln(a.Err, err)
			return exitFailure
		}
		if running && !force {
			continue
		}
		if !dryRun {
			if err := store.Remove(p.Name); err != nil {
				fmt.Fprintln(a.Err, err)
				return exitFailure
			}
		}
		removed = append(removed, p)
	}
	if flags.JSON {
		b, _ := json.MarshalIndent(removed, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return exitSuccess
	}
	for _, p := range removed {
		fmt.Fprintf(a.Out, "pruned %s\n", p.Name)
	}
	return exitSuccess
}

func (a App) runTabNew(store profile.Store, mgr daemon.Manager, flags GlobalFlags, url string) int {
	name := flags.Profile
	if name == "" {
		fmt.Fprintln(a.Err, "-p/--profile is required")
		return exitUsage
	}
	_, _, err := store.Upsert(name, profile.Overrides{})
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if err := ensureRunning(mgr, name, flags.NoStart); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	client, err := daemon.NewClient(mgr.SocketPath(profile.SafeName(name)))
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()

	tab, err := client.TabNew(url)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	fmt.Fprintf(a.Out, "%d\n", tab.ID)
	return exitSuccess
}

func (a App) runTabList(store profile.Store, mgr daemon.Manager, flags GlobalFlags) int {
	client, err := a.prepareClientNoTab(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	tabs, err := client.TabList()
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if flags.JSON {
		b, _ := json.MarshalIndent(tabs, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return exitSuccess
	}
	for _, tab := range tabs {
		marker := ""
		if tab.Active {
			marker = "*"
		}
		fmt.Fprintf(a.Out, "%d%s %s\n", tab.ID, marker, tab.URL)
	}
	return exitSuccess
}

func (a App) runTabClose(store profile.Store, mgr daemon.Manager, flags GlobalFlags, tab int) int {
	client, err := a.prepareClientNoTab(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	if err := client.TabClose(tab); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	return exitSuccess
}

func (a App) runTabSwitch(store profile.Store, mgr daemon.Manager, flags GlobalFlags, tab int) int {
	client, err := a.prepareClientNoTab(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	if err := client.TabSwitch(tab); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	return exitSuccess
}

func (a App) runGoto(store profile.Store, mgr daemon.Manager, flags GlobalFlags, url string) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	timeoutMs, err := actionTimeoutMs(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	if err := client.Goto(tabID, url, timeoutMs); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	_, _ = store.Touch(flags.Profile)
	return exitSuccess
}

func (a App) runClick(store profile.Store, mgr daemon.Manager, flags GlobalFlags, selector string) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	timeoutMs, err := actionTimeoutMs(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	if err := client.Click(tabID, normalizeSelector(selector), timeoutMs); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	_, _ = store.Touch(flags.Profile)
	return exitSuccess
}

func (a App) runFill(store profile.Store, mgr daemon.Manager, flags GlobalFlags, selector string, value string) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	timeoutMs, err := actionTimeoutMs(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	if err := client.Fill(tabID, normalizeSelector(selector), value, timeoutMs); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	_, _ = store.Touch(flags.Profile)
	return exitSuccess
}

func (a App) runShot(store profile.Store, mgr daemon.Manager, flags GlobalFlags, path string, fullPage bool, selector string) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	timeoutMs, err := actionTimeoutMs(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	if err := client.Shot(tabID, path, fullPage, selector, timeoutMs); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	_, _ = store.Touch(flags.Profile)
	return exitSuccess
}

func (a App) runExtract(store profile.Store, mgr daemon.Manager, flags GlobalFlags) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	timeoutMs, err := actionTimeoutMs(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	result, err := client.ExtractWithOptions(tabID, flags.Selector, flags.Main, timeoutMs)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if flags.JSON {
		fmt.Fprintln(a.Out, string(result))
		return exitSuccess
	}
	fmt.Fprintln(a.Out, string(result))
	_, _ = store.Touch(flags.Profile)
	return exitSuccess
}

func (a App) runRead(store profile.Store, mgr daemon.Manager, flags GlobalFlags) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	timeoutMs, err := actionTimeoutMs(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	result, err := client.ExtractWithOptions(tabID, flags.Selector, flags.Main, timeoutMs)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	fmt.Fprintln(a.Out, parsed.Text)
	_, _ = store.Touch(flags.Profile)
	return exitSuccess
}

func (a App) runURL(store profile.Store, mgr daemon.Manager, flags GlobalFlags) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	value, err := client.URL(tabID)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	fmt.Fprintln(a.Out, value)
	return exitSuccess
}

func (a App) runLinks(store profile.Store, mgr daemon.Manager, flags GlobalFlags, filter string) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	links, err := client.Links(tabID, filter)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	if flags.JSON {
		b, _ := json.MarshalIndent(links, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return exitSuccess
	}
	for _, link := range links {
		fmt.Fprintf(a.Out, "%s\t%s\n", link.Text, link.Href)
	}
	return exitSuccess
}

func (a App) runEval(store profile.Store, mgr daemon.Manager, flags GlobalFlags, js string) int {
	client, tabID, err := a.prepareClient(store, mgr, flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	defer client.Close()
	timeoutMs, err := actionTimeoutMs(flags)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitUsage
	}
	result, err := client.Eval(tabID, js, timeoutMs)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	fmt.Fprintln(a.Out, string(result))
	_, _ = store.Touch(flags.Profile)
	return exitSuccess
}

func (a App) prepareClient(store profile.Store, mgr daemon.Manager, flags GlobalFlags) (*daemon.Client, int, error) {
	name := flags.Profile
	if name == "" {
		return nil, 0, errors.New("-p/--profile is required")
	}
	if _, _, err := store.Upsert(name, profile.Overrides{}); err != nil {
		return nil, 0, err
	}
	if err := ensureRunning(mgr, name, flags.NoStart); err != nil {
		return nil, 0, err
	}
	client, err := daemon.NewClient(mgr.SocketPath(profile.SafeName(name)))
	if err != nil {
		return nil, 0, err
	}
	tabID, err := resolveTabID(client, flags.Tab)
	if err != nil {
		_ = client.Close()
		return nil, 0, err
	}
	return client, tabID, nil
}

func (a App) prepareClientNoTab(store profile.Store, mgr daemon.Manager, flags GlobalFlags) (*daemon.Client, error) {
	name := flags.Profile
	if name == "" {
		return nil, errors.New("-p/--profile is required")
	}
	if _, _, err := store.Upsert(name, profile.Overrides{}); err != nil {
		return nil, err
	}
	if err := ensureRunning(mgr, name, flags.NoStart); err != nil {
		return nil, err
	}
	client, err := daemon.NewClient(mgr.SocketPath(profile.SafeName(name)))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func actionTimeoutMs(flags GlobalFlags) (int, error) {
	if strings.TrimSpace(flags.Timeout) == "" {
		return int((20 * time.Second).Milliseconds()), nil
	}
	d, err := time.ParseDuration(flags.Timeout)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout: %w", err)
	}
	if d <= 0 {
		return 0, nil
	}
	return int(d.Milliseconds()), nil
}

func (a App) runServe(store profile.Store, flags GlobalFlags) int {
	name := flags.Profile
	if name == "" {
		fmt.Fprintln(a.Err, "-p/--profile is required")
		return exitUsage
	}
	p, err := store.Load(name)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	socket := filepath.Join(store.ProfileDir(p.Name), "daemon.sock")
	info := daemon.Info{PID: os.Getpid(), Socket: socket, StartedAt: daemon.NowUTC()}
	if path, modTime, err := daemon.CurrentBinaryInfo(); err == nil {
		info.BinaryPath = path
		info.BinaryModTime = modTime
	}
	if err := daemon.WriteInfo(filepath.Join(store.ProfileDir(p.Name), "daemon.json"), info); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	opts := browser.StartOptions{Browser: p.Browser, Channel: p.Channel, Headless: p.Headless, StorageIn: store.StorageStatePath(p.Name)}
	if err := daemon.ServeProfile(socket, p.Name, browser.PlaywrightEngine{}, opts); err != nil {
		fmt.Fprintln(a.Err, err)
		return exitFailure
	}
	return exitSuccess
}

func ensureRunning(mgr daemon.Manager, name string, noStart bool) error {
	running, _, err := mgr.IsRunning(name)
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	if noStart {
		return errors.New("profile is not running")
	}
	return mgr.Start(name)
}

func resolveTabID(client *daemon.Client, requested int) (int, error) {
	if requested != 0 {
		return requested, nil
	}
	status, err := client.Status()
	if err != nil {
		return 0, err
	}
	return resolveTabIDFromStatus(status)
}

func resolveTabIDFromStatus(status daemon.StatusResult) (int, error) {
	if len(status.Tabs) == 1 {
		return status.Tabs[0].ID, nil
	}
	if len(status.Tabs) == 0 {
		return 0, errors.New("no tabs available")
	}
	return 0, errors.New("multiple tabs; use --tab")
}

func normalizeSelector(value string) string {
	if strings.HasPrefix(value, "text=") {
		return value
	}
	if strings.HasPrefix(value, "css=") {
		return value
	}
	return "text=" + value
}

func overridesFromFlags(flags GlobalFlags) (profile.Overrides, error) {
	var overrides profile.Overrides
	if flags.Browser != "" {
		overrides.Browser = flags.Browser
	}
	if flags.Channel != "" {
		overrides.Channel = flags.Channel
	}
	if flags.Headless && flags.Headed {
		return overrides, errors.New("cannot set both --headless and --headed")
	}
	if flags.Headless {
		headless := true
		overrides.Headless = &headless
	}
	if flags.Headed {
		headless := false
		overrides.Headless = &headless
	}
	if flags.TTL != "" {
		d, err := time.ParseDuration(flags.TTL)
		if err != nil {
			return overrides, fmt.Errorf("invalid ttl: %w", err)
		}
		overrides.TTL = &d
	}
	return overrides, nil
}
