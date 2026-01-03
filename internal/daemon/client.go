package daemon

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/patrickjm/www/internal/browser"
)

type Client struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
}

var reqCounter uint64

func NewClient(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, enc: json.NewEncoder(conn), dec: json.NewDecoder(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Call(method string, params any, out any) error {
	id := strconv.FormatUint(atomic.AddUint64(&reqCounter, 1), 10)
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		raw = b
	}
	if err := c.enc.Encode(Request{ID: id, Method: method, Params: raw}); err != nil {
		return err
	}
	var resp Response
	if err := c.dec.Decode(&resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return errors.New(resp.Error.Message)
	}
	if out != nil {
		return json.Unmarshal(resp.Result, out)
	}
	return nil
}

func (c *Client) Status() (StatusResult, error) {
	var result StatusResult
	return result, c.Call("Status", nil, &result)
}

func (c *Client) TabList() ([]TabInfo, error) {
	var result []TabInfo
	return result, c.Call("TabList", nil, &result)
}

func (c *Client) TabNew(url string) (TabInfo, error) {
	var result TabInfo
	return result, c.Call("TabNew", TabNewParams{URL: url}, &result)
}

func (c *Client) TabSwitch(tab int) error {
	return c.Call("TabSwitch", TabSwitchParams{Tab: tab}, nil)
}

func (c *Client) TabClose(tab int) error {
	return c.Call("TabClose", TabCloseParams{Tab: tab}, nil)
}

func (c *Client) Goto(tab int, url string, timeoutMs int) error {
	return c.Call("Goto", GotoParams{Tab: tab, URL: url, TimeoutMs: timeoutMs}, nil)
}

func (c *Client) Click(tab int, selector string, timeoutMs int) error {
	return c.Call("Click", ClickParams{Tab: tab, Selector: selector, TimeoutMs: timeoutMs}, nil)
}

func (c *Client) Fill(tab int, selector string, value string, timeoutMs int) error {
	return c.Call("Fill", FillParams{Tab: tab, Selector: selector, Value: value, TimeoutMs: timeoutMs}, nil)
}

func (c *Client) Shot(tab int, path string, fullPage bool, selector string, timeoutMs int) error {
	return c.Call("Shot", ShotParams{Tab: tab, Path: path, FullPage: fullPage, Selector: selector, TimeoutMs: timeoutMs}, nil)
}

func (c *Client) Extract(tab int, timeoutMs int) (json.RawMessage, error) {
	var result json.RawMessage
	return result, c.Call("Extract", ExtractParams{Tab: tab, TimeoutMs: timeoutMs}, &result)
}

func (c *Client) ExtractWithOptions(tab int, selector string, main bool, timeoutMs int) (json.RawMessage, error) {
	var result json.RawMessage
	return result, c.Call("Extract", ExtractParams{Tab: tab, Selector: selector, Main: main, TimeoutMs: timeoutMs}, &result)
}

func (c *Client) Eval(tab int, js string, timeoutMs int) (json.RawMessage, error) {
	var result json.RawMessage
	return result, c.Call("Eval", EvalParams{Tab: tab, JS: js, TimeoutMs: timeoutMs}, &result)
}

func (c *Client) Stop() error {
	return c.Call("Stop", nil, nil)
}

func (c *Client) URL(tab int) (string, error) {
	var result string
	return result, c.Call("URL", URLParams{Tab: tab}, &result)
}

func (c *Client) Links(tab int, filter string) ([]browser.ExtractLink, error) {
	var result []browser.ExtractLink
	return result, c.Call("Links", LinksParams{Tab: tab, Filter: filter}, &result)
}
