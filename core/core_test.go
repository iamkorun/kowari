package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureHandler(t *testing.T) {
	s := NewStore("")
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := strings.NewReader(`{"hello":"world"}`)
	req, _ := http.NewRequest("POST", ts.URL+"/webhook?x=1", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom", "kowari")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("want 1 captured, got %d", len(list))
	}
	got := list[0]
	if got.Method != "POST" {
		t.Errorf("method = %q", got.Method)
	}
	if got.Path != "/webhook?x=1" {
		t.Errorf("path = %q", got.Path)
	}
	if string(got.Body) != `{"hello":"world"}` {
		t.Errorf("body = %q", got.Body)
	}
	if got.Headers.Get("X-Custom") != "kowari" {
		t.Errorf("missing X-Custom header")
	}
	if got.ID != 1 {
		t.Errorf("id = %d", got.ID)
	}
}

func TestReplay(t *testing.T) {
	var (
		gotMethod string
		gotBody   []byte
		gotHeader string
	)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotBody, _ = io.ReadAll(r.Body)
		gotHeader = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer target.Close()

	r := &Request{
		Method:  "PUT",
		Path:    "/forward",
		Headers: http.Header{"X-Custom": []string{"kowari"}, "Content-Type": []string{"application/json"}},
		Body:    []byte(`{"a":1}`),
	}
	code, err := Replay(nil, target.URL, r)
	if err != nil {
		t.Fatal(err)
	}
	if code != http.StatusAccepted {
		t.Errorf("code = %d", code)
	}
	if gotMethod != "PUT" {
		t.Errorf("target method = %q", gotMethod)
	}
	if !bytes.Equal(gotBody, r.Body) {
		t.Errorf("body mismatch: %q", gotBody)
	}
	if gotHeader != "kowari" {
		t.Errorf("header mismatch: %q", gotHeader)
	}
}

func TestJSONLPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")
	s := NewStore(path)

	for i := 0; i < 3; i++ {
		s.Add(&Request{
			Method:  "POST",
			Path:    "/h",
			Headers: http.Header{"Content-Type": []string{"application/json"}},
			Body:    []byte(`{}`),
		})
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	n := 0
	for scan.Scan() {
		var r Request
		if err := json.Unmarshal(scan.Bytes(), &r); err != nil {
			t.Fatalf("line %d not valid json: %v", n, err)
		}
		if r.Method == "" || r.Path == "" || r.Timestamp.IsZero() {
			t.Errorf("missing fields on line %d: %+v", n, r)
		}
		n++
	}
	if n != 3 {
		t.Errorf("want 3 lines, got %d", n)
	}
}

func TestClearAndSetReplayCode(t *testing.T) {
	s := NewStore("")
	r := s.Add(&Request{Method: "GET", Path: "/"})
	s.SetReplayCode(r.ID, 201)
	if s.List()[0].ReplayCode != 201 {
		t.Errorf("replay code not set")
	}
	s.Clear()
	if len(s.List()) != 0 {
		t.Errorf("clear failed")
	}
}
