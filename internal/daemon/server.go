package daemon

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/patrickjm/www/internal/browser"
)

type Server struct {
	profile     string
	engine      browser.Engine
	storagePath string
	mu          sync.Mutex
	session     browser.Session
	tabs        map[int]browser.Page
	activeTab   int
	nextTabID   int
	stop        chan struct{}
	stopOnce    sync.Once
}

func NewServer(profile string, engine browser.Engine, storagePath string) *Server {
	return &Server{
		profile:     profile,
		engine:      engine,
		storagePath: storagePath,
		tabs:        make(map[int]browser.Page),
		nextTabID:   1,
		stop:        make(chan struct{}),
	}
}

func (s *Server) Init(opts browser.StartOptions) error {
	session, err := s.engine.Start(opts)
	if err != nil {
		return err
	}
	s.session = session
	page, err := session.NewPage()
	if err != nil {
		return err
	}
	s.tabs[1] = page
	s.activeTab = 1
	s.nextTabID = 2
	return nil
}

func (s *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-s.stop:
				return nil
			default:
			}
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			return
		}
		resp := s.handleRequest(req)
		_ = enc.Encode(resp)
		if req.Method == "Stop" {
			return
		}
	}
}

func (s *Server) handleRequest(req Request) Response {
	result, err := s.dispatch(req)
	if err != nil {
		return Response{ID: req.ID, Error: &RespError{Message: err.Error()}}
	}
	if result == nil {
		return Response{ID: req.ID}
	}
	b, err := json.Marshal(result)
	if err != nil {
		return Response{ID: req.ID, Error: &RespError{Message: err.Error()}}
	}
	return Response{ID: req.ID, Result: b}
}

func (s *Server) dispatch(req Request) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch req.Method {
	case "Status":
		return s.statusLocked()
	case "TabList":
		return s.statusLockedTabs()
	case "TabNew":
		var params TabNewParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return s.tabNewLocked(params.URL)
	case "TabSwitch":
		var params TabSwitchParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return nil, s.tabSwitchLocked(params.Tab)
	case "TabClose":
		var params TabCloseParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return nil, s.tabCloseLocked(params.Tab)
	case "Goto":
		var params GotoParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return nil, s.withTabLockedTimeout(params.Tab, params.TimeoutMs, func(p browser.Page) error {
			return p.Goto(params.URL)
		})
	case "Click":
		var params ClickParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return nil, s.withTabLockedTimeout(params.Tab, params.TimeoutMs, func(p browser.Page) error {
			return p.Click(params.Selector)
		})
	case "Fill":
		var params FillParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return nil, s.withTabLockedTimeout(params.Tab, params.TimeoutMs, func(p browser.Page) error {
			return p.Fill(params.Selector, params.Value)
		})
	case "Shot":
		var params ShotParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return nil, s.withTabLockedTimeout(params.Tab, params.TimeoutMs, func(p browser.Page) error {
			return p.Screenshot(params.Path, params.FullPage, params.Selector)
		})
	case "Extract":
		var params ExtractParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		var result browser.ExtractResult
		if err := s.withTabLockedTimeout(params.Tab, params.TimeoutMs, func(p browser.Page) error {
			var err error
			result, err = p.Extract(browser.ExtractOptions{Selector: params.Selector, Main: params.Main})
			return err
		}); err != nil {
			return nil, err
		}
		return result, nil
	case "URL":
		var params URLParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		var value string
		if err := s.withTabLocked(params.Tab, func(p browser.Page) error {
			var err error
			value, err = p.URL()
			return err
		}); err != nil {
			return nil, err
		}
		return value, nil
	case "Links":
		var params LinksParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		var links []browser.ExtractLink
		if err := s.withTabLocked(params.Tab, func(p browser.Page) error {
			var err error
			links, err = p.Links(params.Filter)
			return err
		}); err != nil {
			return nil, err
		}
		return links, nil
	case "Eval":
		var params EvalParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		var result json.RawMessage
		if err := s.withTabLockedTimeout(params.Tab, params.TimeoutMs, func(p browser.Page) error {
			var err error
			result, err = p.Eval(params.JS)
			return err
		}); err != nil {
			return nil, err
		}
		return result, nil
	case "Stop":
		_ = s.persistStorageLocked()
		_ = s.shutdownLocked()
		s.stopOnce.Do(func() { close(s.stop) })
		return nil, nil
	default:
		return nil, errors.New("unknown method")
	}
}

func (s *Server) statusLocked() (StatusResult, error) {
	tabs, err := s.statusLockedTabs()
	if err != nil {
		return StatusResult{}, err
	}
	return StatusResult{Profile: s.profile, Tabs: tabs}, nil
}

func (s *Server) statusLockedTabs() ([]TabInfo, error) {
	infos := make([]TabInfo, 0, len(s.tabs))
	for id, page := range s.tabs {
		url, _ := page.URL()
		title, _ := page.Title()
		infos = append(infos, TabInfo{ID: id, URL: url, Title: title, Active: id == s.activeTab})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos, nil
}

func (s *Server) tabNewLocked(url string) (TabInfo, error) {
	page, err := s.session.NewPage()
	if err != nil {
		return TabInfo{}, err
	}
	id := s.nextTabID
	s.nextTabID++
	s.tabs[id] = page
	s.activeTab = id
	if url != "" {
		if err := page.Goto(url); err != nil {
			return TabInfo{}, err
		}
	}
	_ = s.persistStorageLocked()
	return TabInfo{ID: id, Active: true}, nil
}

func (s *Server) tabSwitchLocked(tab int) error {
	if _, ok := s.tabs[tab]; !ok {
		return errors.New("tab not found")
	}
	s.activeTab = tab
	return nil
}

func (s *Server) tabCloseLocked(tab int) error {
	page, ok := s.tabs[tab]
	if !ok {
		return errors.New("tab not found")
	}
	_ = page.Close()
	delete(s.tabs, tab)
	if s.activeTab == tab {
		s.activeTab = 0
		for id := range s.tabs {
			s.activeTab = id
			break
		}
	}
	_ = s.persistStorageLocked()
	return nil
}

func (s *Server) withTabLocked(tab int, fn func(browser.Page) error) error {
	if tab == 0 {
		tab = s.activeTab
	}
	page, ok := s.tabs[tab]
	if !ok {
		return errors.New("tab not found")
	}
	if err := fn(page); err != nil {
		return err
	}
	return s.persistStorageLocked()
}

func (s *Server) withTabLockedTimeout(tab int, timeoutMs int, fn func(browser.Page) error) error {
	return s.withTabLocked(tab, func(p browser.Page) error {
		if timeoutMs > 0 {
			_ = p.SetTimeout(timeoutMs)
		}
		return fn(p)
	})
}

func (s *Server) persistStorageLocked() error {
	if s.storagePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	return s.session.StorageState(s.storagePath)
}

func (s *Server) shutdownLocked() error {
	if s.session != nil {
		return s.session.Close()
	}
	return nil
}

func ServeProfile(socketPath string, profile string, engine browser.Engine, opts browser.StartOptions) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return err
	}
	server := NewServer(profile, engine, opts.StorageIn)
	if err := server.Init(opts); err != nil {
		return err
	}
	if err := os.RemoveAll(socketPath); err != nil {
		_ = server.shutdownLocked()
		return err
	}
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = server.shutdownLocked()
		return err
	}
	defer l.Close()
	go func() {
		<-server.stop
		_ = l.Close()
	}()
	return server.Serve(l)
}

func WriteInfo(path string, info Info) error {
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}
