# www

Playwright-based CLI for persistent browser profiles and tab-level automation.

## Quick start

```sh
www install
www start -p demo
www -p demo goto https://example.com
www -p demo url
www -p demo links --filter "More"
www -p demo read --main
www -p demo stop
```

## Install

### Homebrew (planned)
This repo ships a Homebrew formula template. To distribute via brew, publish a release and install from the tap:

```sh
brew install --HEAD patrickjm/tap/www
```

### Go install
Install with:

```sh
go install github.com/patrickjm/www/cmd/www@latest
```

## Commands

- `www install`
- `www doctor`
- `www start -p NAME`
- `www stop -p NAME`
- `www ps`
- `www list`
- `www show NAME`
- `www rm NAME...`
- `www prune [--dry-run] [--force]`
- `www tab new -p NAME [--url URL]`
- `www tab list -p NAME`
- `www tab close -p NAME --tab ID`
- `www tab switch -p NAME --tab ID`
- `www goto -p NAME URL`
- `www click -p NAME TEXT|SELECTOR`
- `www fill -p NAME SELECTOR VALUE`
- `www shot -p NAME PATH [--full-page] [--selector SELECTOR]`
- `www extract -p NAME [--main] [--selector SELECTOR] [--json]`
- `www read -p NAME [--main] [--selector SELECTOR]`
- `www url -p NAME`
- `www links -p NAME [--filter TEXT] [--json]`
- `www eval -p NAME JS`

## Configuration

System config (TOML):
- `/opt/homebrew/etc/www/config.toml`
- `/usr/local/etc/www/config.toml`

Env vars:
- `WWW_PROFILE_DIR`
- `WWW_DEFAULT_TTL`

Default profile directory:
- macOS: `~/Library/Application Support/www`
- Linux: `$XDG_DATA_HOME/www` or `~/.local/share/www`

Timeouts:
- Default action timeout is `20s`
- Override with `-t/--timeout 60s`

## Notes

- Profiles auto-create on first use.
- Tabs are explicit; when multiple tabs exist, use `--tab`.
- Headless is the default.
