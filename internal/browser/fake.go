package browser

import (
	"encoding/json"
	"errors"
)

type FakeEngine struct {
	Session *FakeSession
}

func (f *FakeEngine) Start(opts StartOptions) (Session, error) {
	if f.Session == nil {
		f.Session = &FakeSession{}
	}
	return f.Session, nil
}

type FakeSession struct {
	Pages       []*FakePage
	Closed      bool
	StoragePath string
}

func (s *FakeSession) NewPage() (Page, error) {
	page := &FakePage{TitleValue: "", URLValue: ""}
	s.Pages = append(s.Pages, page)
	return page, nil
}

func (s *FakeSession) Close() error {
	s.Closed = true
	return nil
}

func (s *FakeSession) StorageState(path string) error {
	s.StoragePath = path
	return nil
}

type FakePage struct {
	URLValue   string
	TitleValue string
	Clicks     []string
	Fills      []string
	Shots      []string
	EvalResult json.RawMessage
	ExtractRes ExtractResult
	LinksRes   []ExtractLink
	TimeoutMs  int
	Closed     bool
}

func (p *FakePage) Goto(url string) error {
	p.URLValue = url
	return nil
}

func (p *FakePage) Click(selector string) error {
	p.Clicks = append(p.Clicks, selector)
	return nil
}

func (p *FakePage) Fill(selector string, value string) error {
	p.Fills = append(p.Fills, selector+"="+value)
	return nil
}

func (p *FakePage) Screenshot(path string, fullPage bool, selector string) error {
	p.Shots = append(p.Shots, path)
	return nil
}

func (p *FakePage) Extract(_ ExtractOptions) (ExtractResult, error) {
	if p.ExtractRes.URL != "" || p.ExtractRes.Title != "" || p.ExtractRes.Text != "" {
		return p.ExtractRes, nil
	}
	return ExtractResult{URL: p.URLValue, Title: p.TitleValue, Text: ""}, nil
}

func (p *FakePage) Links(_ string) ([]ExtractLink, error) {
	return p.LinksRes, nil
}

func (p *FakePage) SetTimeout(ms int) error {
	p.TimeoutMs = ms
	return nil
}

func (p *FakePage) Eval(js string) (json.RawMessage, error) {
	if p.EvalResult == nil {
		return nil, errors.New("no eval result")
	}
	return p.EvalResult, nil
}

func (p *FakePage) URL() (string, error) {
	return p.URLValue, nil
}

func (p *FakePage) Title() (string, error) {
	return p.TitleValue, nil
}

func (p *FakePage) Close() error {
	p.Closed = true
	return nil
}
