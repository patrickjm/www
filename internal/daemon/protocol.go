package daemon

import "encoding/json"

type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RespError      `json:"error,omitempty"`
}

type RespError struct {
	Message string `json:"message"`
}

type TabInfo struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Title  string `json:"title"`
	Active bool   `json:"active"`
}

type StatusResult struct {
	Profile string    `json:"profile"`
	Tabs    []TabInfo `json:"tabs"`
}

type TabNewParams struct {
	URL string `json:"url,omitempty"`
}

type TabSwitchParams struct {
	Tab int `json:"tab"`
}

type TabCloseParams struct {
	Tab int `json:"tab"`
}

type GotoParams struct {
	Tab       int    `json:"tab"`
	URL       string `json:"url"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type ClickParams struct {
	Tab       int    `json:"tab"`
	Selector  string `json:"selector"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type FillParams struct {
	Tab       int    `json:"tab"`
	Selector  string `json:"selector"`
	Value     string `json:"value"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type ShotParams struct {
	Tab       int    `json:"tab"`
	Path      string `json:"path"`
	FullPage  bool   `json:"full_page"`
	Selector  string `json:"selector,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type ExtractParams struct {
	Tab       int    `json:"tab"`
	Selector  string `json:"selector,omitempty"`
	Main      bool   `json:"main,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type EvalParams struct {
	Tab       int    `json:"tab"`
	JS        string `json:"js"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type URLParams struct {
	Tab int `json:"tab"`
}

type LinksParams struct {
	Tab    int    `json:"tab"`
	Filter string `json:"filter,omitempty"`
}
