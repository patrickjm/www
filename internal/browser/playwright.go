package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type PlaywrightEngine struct{}

func (p PlaywrightEngine) Start(opts StartOptions) (Session, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}
	bt, err := browserType(pw, opts.Browser)
	if err != nil {
		pw.Stop()
		return nil, err
	}
	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(opts.Headless),
	}
	if opts.Channel != "" {
		launchOpts.Channel = playwright.String(opts.Channel)
	}
	browser, err := bt.Launch(launchOpts)
	if err != nil {
		pw.Stop()
		return nil, err
	}
	ctxOpts := playwright.BrowserNewContextOptions{}
	if opts.StorageIn != "" {
		if _, err := os.Stat(opts.StorageIn); err == nil {
			ctxOpts.StorageStatePath = playwright.String(opts.StorageIn)
		}
	}
	ctx, err := browser.NewContext(ctxOpts)
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, err
	}
	return &playwrightSession{pw: pw, browser: browser, ctx: ctx}, nil
}

type playwrightSession struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	ctx     playwright.BrowserContext
}

func (s *playwrightSession) NewPage() (Page, error) {
	page, err := s.ctx.NewPage()
	if err != nil {
		return nil, err
	}
	return &playwrightPage{page: page}, nil
}

func (s *playwrightSession) StorageState(path string) error {
	_, err := s.ctx.StorageState(path)
	return err
}

func (s *playwrightSession) Close() error {
	if s.ctx != nil {
		_ = s.ctx.Close()
	}
	if s.browser != nil {
		_ = s.browser.Close()
	}
	if s.pw != nil {
		s.pw.Stop()
	}
	return nil
}

type playwrightPage struct {
	page playwright.Page
}

func (p *playwrightPage) Goto(url string) error {
	_, err := p.page.Goto(url)
	return err
}

func (p *playwrightPage) Click(selector string) error {
	if strings.HasPrefix(selector, "text=") {
		return p.clickByText(strings.TrimPrefix(selector, "text="))
	}
	return p.page.Click(selector)
}

func (p *playwrightPage) Fill(selector string, value string) error {
	return p.page.Fill(selector, value)
}

func (p *playwrightPage) Screenshot(path string, fullPage bool, selector string) error {
	if selector != "" {
		locator := p.page.Locator(selector)
		_, err := locator.Screenshot(playwright.LocatorScreenshotOptions{Path: playwright.String(path)})
		return err
	}
	_, err := p.page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(path), FullPage: playwright.Bool(fullPage)})
	return err
}

func (p *playwrightPage) Extract(options ExtractOptions) (ExtractResult, error) {
	var result ExtractResult
	v, err := p.page.Evaluate(`(opts) => {
  const selector = opts && opts.selector ? String(opts.selector) : "";
  const main = opts && opts.main;
  const pickRoot = () => {
    if (selector) return document.querySelector(selector);
    if (!main) return document.body;
    const preferred = document.querySelector("[role=main]") || document.querySelector("main") || document.querySelector("article");
    if (preferred) return preferred;
    const candidates = Array.from(document.querySelectorAll("main, article, [role=main], #content, .content, .docs-content, .markdown, .markdown-body, section"));
    let best = null;
    let bestLen = 0;
    for (const el of candidates) {
      const len = (el.textContent || "").trim().length;
      if (len > bestLen) {
        bestLen = len;
        best = el;
      }
    }
    return best || document.body;
  };
  let root = pickRoot();
  let text = root ? (root.innerText || root.textContent || "") : "";
  if (main && (!text || !text.trim()) && root !== document.body) {
    root = document.body;
    text = root ? (root.innerText || root.textContent || "") : "";
  }
  const links = Array.from(document.querySelectorAll("a")).map(a => ({ text: a.innerText || "", href: a.href || "" }));
  const buttons = Array.from(document.querySelectorAll("button, [role=button]")).map(b => ({ text: b.innerText || "" }));
  const inputs = Array.from(document.querySelectorAll("input, textarea, select")).map(i => ({
    label: i.labels && i.labels.length ? i.labels[0].innerText || "" : "",
    name: i.name || "",
    type: i.type || i.tagName.toLowerCase(),
  }));
  const meta = {};
  document.querySelectorAll('meta[name]').forEach(m => { meta[m.name] = m.content || ""; });
  return { url: location.href, title: document.title || "", text, links, buttons, inputs, meta };
}`, map[string]any{"selector": options.Selector, "main": options.Main})
	if err != nil {
		return result, err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (p *playwrightPage) Links(filter string) ([]ExtractLink, error) {
	value, err := p.page.Evaluate(`(f) => {
  const filter = f ? String(f).toLowerCase() : "";
  const links = Array.from(document.querySelectorAll("a")).map(a => ({
    text: (a.innerText || "").trim(),
    href: a.href || ""
  })).filter(l => l.text && l.href);
  if (!filter) return links;
  return links.filter(l => l.text.toLowerCase().includes(filter));
}`, filter)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var links []ExtractLink
	if err := json.Unmarshal(b, &links); err != nil {
		return nil, err
	}
	return links, nil
}

func (p *playwrightPage) SetTimeout(ms int) error {
	if ms <= 0 {
		return nil
	}
	p.page.SetDefaultTimeout(float64(ms))
	return nil
}

func (p *playwrightPage) Eval(js string) (json.RawMessage, error) {
	v, err := p.page.Evaluate(js)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (p *playwrightPage) URL() (string, error) {
	return p.page.URL(), nil
}

func (p *playwrightPage) Title() (string, error) {
	return p.page.Title()
}

func (p *playwrightPage) Close() error {
	return p.page.Close()
}

func (p *playwrightPage) clickByText(text string) error {
	if err := p.page.GetByText(text, playwright.PageGetByTextOptions{Exact: playwright.Bool(true)}).Click(); err == nil {
		return nil
	}
	if err := p.page.GetByText(text, playwright.PageGetByTextOptions{Exact: playwright.Bool(false)}).Click(); err == nil {
		return nil
	}
	escaped := strings.ReplaceAll(text, "\"", "\\\"")
	selectors := []string{
		fmt.Sprintf("a:has-text(\"%s\")", escaped),
		fmt.Sprintf("button:has-text(\"%s\")", escaped),
		fmt.Sprintf("[role=button]:has-text(\"%s\")", escaped),
		fmt.Sprintf("label:has-text(\"%s\")", escaped),
	}
	for _, sel := range selectors {
		if err := p.page.Locator(sel).First().Click(); err == nil {
			return nil
		}
	}
	suggestion, sErr := p.suggestText(text)
	if sErr == nil && suggestion != "" {
		return fmt.Errorf("no match for text=%q. did you mean %q?", text, suggestion)
	}
	return fmt.Errorf("no match for text=%q", text)
}

func (p *playwrightPage) suggestText(text string) (string, error) {
	value, err := p.page.Evaluate(`() => {
  const candidates = new Set();
  const pushText = (t) => {
    if (!t) return;
    const v = String(t).trim();
    if (v) candidates.add(v);
  };
  document.querySelectorAll("a,button,[role=button],input[type=submit],input[type=button],label,[aria-label]").forEach(el => {
    pushText(el.innerText);
    if (el.getAttribute) pushText(el.getAttribute("aria-label"));
    if (el.value) pushText(el.value);
  });
  return Array.from(candidates).slice(0, 200);
}`)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var candidates []string
	if err := json.Unmarshal(b, &candidates); err != nil {
		return "", err
	}
	query := normalizeText(text)
	if query == "" {
		return "", nil
	}
	best := ""
	bestScore := -1
	for _, candidate := range candidates {
		normalized := normalizeText(candidate)
		if normalized == "" {
			continue
		}
		score := levenshteinDistance(query, normalized)
		if bestScore == -1 || score < bestScore {
			bestScore = score
			best = candidate
		}
	}
	return best, nil
}

func normalizeText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = minInt(del, ins, sub)
		}
		prev = cur
	}
	return prev[len(b)]
}

func minInt(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func browserType(pw *playwright.Playwright, name string) (playwright.BrowserType, error) {
	switch name {
	case "chromium", "":
		return pw.Chromium, nil
	case "firefox":
		return pw.Firefox, nil
	case "webkit":
		return pw.WebKit, nil
	default:
		return nil, errors.New("unknown browser: " + name)
	}
}
