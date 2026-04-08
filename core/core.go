// Package core contains the TUI-independent webhook capture/replay logic.
package core

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Request is a captured HTTP request.
type Request struct {
	ID         int         `json:"id"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
	Timestamp  time.Time   `json:"timestamp"`
	ReplayCode int         `json:"replay_code,omitempty"`
}

// Store is a thread-safe in-memory list of captured requests.
type Store struct {
	mu       sync.RWMutex
	reqs     []*Request
	nextID   int
	savePath string
	onAdd    func(*Request)
}

// NewStore creates a new store. If savePath is non-empty, each captured
// request is appended as a JSON line to the file.
func NewStore(savePath string) *Store {
	return &Store{savePath: savePath}
}

// OnAdd registers a callback fired after a request is captured.
func (s *Store) OnAdd(fn func(*Request)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onAdd = fn
}

// Add appends a request and returns it with an assigned ID.
func (s *Store) Add(r *Request) *Request {
	s.mu.Lock()
	s.nextID++
	r.ID = s.nextID
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	s.reqs = append(s.reqs, r)
	cb := s.onAdd
	path := s.savePath
	s.mu.Unlock()

	if path != "" {
		_ = appendJSONL(path, r)
	}
	if cb != nil {
		cb(r)
	}
	return r
}

// List returns a copy of the captured requests slice.
func (s *Store) List() []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Request, len(s.reqs))
	copy(out, s.reqs)
	return out
}

// Clear removes all captured requests.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs = nil
}

// SetReplayCode updates the replay status for a request by ID.
func (s *Store) SetReplayCode(id, code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.reqs {
		if r.ID == id {
			r.ReplayCode = code
			return
		}
	}
}

// Handler returns an http.Handler that captures every incoming request.
func (s *Store) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		hdr := req.Header.Clone()
		s.Add(&Request{
			Method:  req.Method,
			Path:    req.URL.RequestURI(),
			Headers: hdr,
			Body:    body,
		})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
}

// Replay forwards the captured request to targetURL and returns status code.
func Replay(client *http.Client, targetURL string, r *Request) (int, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequest(r.Method, targetURL+r.Path, bytes.NewReader(r.Body))
	if err != nil {
		return 0, err
	}
	for k, vs := range r.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func appendJSONL(path string, r *Request) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(r)
}
