package browser

import "encoding/json"

type StartOptions struct {
	Browser   string
	Channel   string
	Headless  bool
	StorageIn string
}

type Engine interface {
	Start(opts StartOptions) (Session, error)
}

type Session interface {
	NewPage() (Page, error)
	Close() error
	StorageState(path string) error
}

type Page interface {
	Goto(url string) error
	Click(selector string) error
	Fill(selector string, value string) error
	Screenshot(path string, fullPage bool, selector string) error
	Extract(options ExtractOptions) (ExtractResult, error)
	Links(filter string) ([]ExtractLink, error)
	SetTimeout(ms int) error
	Eval(js string) (json.RawMessage, error)
	URL() (string, error)
	Title() (string, error)
	Close() error
}

type ExtractOptions struct {
	Selector string
	Main     bool
}

type ExtractResult struct {
	URL     string            `json:"url"`
	Title   string            `json:"title"`
	Text    string            `json:"text"`
	Links   []ExtractLink     `json:"links"`
	Buttons []ExtractButton   `json:"buttons"`
	Inputs  []ExtractInput    `json:"inputs"`
	Meta    map[string]string `json:"meta"`
}

type ExtractLink struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

type ExtractButton struct {
	Text string `json:"text"`
}

type ExtractInput struct {
	Label string `json:"label"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}
