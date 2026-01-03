---
name: www-cli
description: Use when you need to drive the www CLI for web navigation, reading/extracting content, listing links, taking screenshots, or automating browser actions with named profiles and tab IDs. Trigger this skill for tasks that require programmatic web access via www (start/stop profiles, goto, click, read/extract, links, url, eval, screenshots, tab management, or timeouts).
---

# www CLI

## Overview

Operate the www CLI to browse, extract, and automate web actions using persistent profiles and explicit tab IDs.

## Quick start

- Start or reuse a profile: `www start -p demo`
- Navigate: `www -p demo goto https://example.com`
- Read main content: `www -p demo read --main`
- List links: `www -p demo links --filter "Docs"`
- Screenshot: `www -p demo shot /tmp/page.png`
- Stop: `www stop -p demo`

## Core workflow

1) Ensure a profile exists (auto-created on first use) and is running.
2) Navigate to the target URL.
3) Use `read` or `extract` to capture content, `links` to list URLs, or `shot` for screenshots.
4) Use `tab` commands when multiple tabs exist.

## Common actions

- **Navigate**: `www -p NAME goto URL`
- **Click**: `www -p NAME click "Text or selector"`
- **Fill**: `www -p NAME fill "Label or selector" "value"`
- **Read**: `www -p NAME read --main` (use `-S/--selector` for custom targets)
- **Extract JSON**: `www -p NAME extract --json --main`
- **List links**: `www -p NAME links --filter "foo"`
- **Screenshot**: `www -p NAME shot /path/out.png -F`
- **URL**: `www -p NAME url`
- **Tabs**: `www -p NAME tab list`, `www -p NAME tab new -u URL`, `www -p NAME tab switch -T 2`

## Flags and defaults

- Profile: `-p/--profile` (required for most commands)
- Tab: `-T/--tab` (required if multiple tabs exist)
- Selector: `-S/--selector` (used by `read`, `extract`, and `shot`)
- Timeout: `-t/--timeout` (default `20s`)
- JSON output: `-j/--json`

## Notes

- Profiles auto-start unless `-N/--no-start` is used.
- Headless is the default; `-E/--headed` overrides.
- Use `links` + `goto` to navigate when clicking is unreliable.
