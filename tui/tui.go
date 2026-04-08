// Package tui renders the kowari split-pane interface.
package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/iamkorun/kowari/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFAA00"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#000")).Background(lipgloss.Color("#FFAA00"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#888"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#3AA655"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5484D"))
)

// tickMsg is emitted periodically to refresh the list.
type tickMsg struct{}

// newReqMsg is sent by the capture callback.
type newReqMsg struct{}

// Model is the Bubble Tea model for the kowari TUI.
type Model struct {
	Store    *core.Store
	Target   string
	Port     int
	cursor   int
	width    int
	height   int
	status   string
	program  *tea.Program
}

// New creates a new Model.
func New(store *core.Store, target string, port int) *Model {
	return &Model{Store: store, Target: target, Port: port}
}

// SetProgram wires a program handle so callbacks can push messages.
func (m *Model) SetProgram(p *tea.Program) { m.program = p }

// Notify pushes a refresh message into the running program (if any).
func (m *Model) Notify() {
	if m.program != nil {
		m.program.Send(newReqMsg{})
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tickMsg, newReqMsg:
		// trigger re-render
	}
	return m, nil
}

func (m *Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := m.Store.List()
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(list)-1 {
			m.cursor++
		}
	case "c":
		m.Store.Clear()
		m.cursor = 0
		m.status = "cleared"
	case "r":
		if len(list) == 0 || m.Target == "" {
			m.status = "no target or no requests"
			break
		}
		req := list[m.cursor]
		code, err := core.Replay(nil, m.Target, req)
		if err != nil {
			m.status = "replay error: " + err.Error()
			m.Store.SetReplayCode(req.ID, 0)
		} else {
			m.status = fmt.Sprintf("replayed #%d → %d", req.ID, code)
			m.Store.SetReplayCode(req.ID, code)
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	list := m.Store.List()
	header := titleStyle.Render(fmt.Sprintf(" 🐨 kowari  ·  :%d → %s ", m.Port, orDash(m.Target)))
	help := dimStyle.Render(" ↑/↓ move · r replay · c clear · q quit ")

	w := m.width
	if w < 40 {
		w = 120
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	contentH := h - 4
	leftW := w/2 - 2
	rightW := w - leftW - 4

	left := borderStyle.Width(leftW).Height(contentH).Render(m.renderList(list, leftW-2, contentH-2))
	right := borderStyle.Width(rightW).Height(contentH).Render(m.renderDetail(list, rightW-2, contentH-2))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	statusLine := dimStyle.Render(m.status)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, help, statusLine)
}

func (m *Model) renderList(list []*core.Request, w, h int) string {
	if len(list) == 0 {
		return dimStyle.Render("waiting for requests…")
	}
	var b strings.Builder
	start := 0
	if m.cursor >= h {
		start = m.cursor - h + 1
	}
	for i := start; i < len(list) && i-start < h; i++ {
		r := list[i]
		status := "  -"
		if r.ReplayCode > 0 {
			if r.ReplayCode >= 200 && r.ReplayCode < 300 {
				status = okStyle.Render(fmt.Sprintf("%3d", r.ReplayCode))
			} else {
				status = errStyle.Render(fmt.Sprintf("%3d", r.ReplayCode))
			}
		}
		line := fmt.Sprintf("#%-3d %-6s %s  %s", r.ID, r.Method, truncate(r.Path, w-25), status)
		line += " " + dimStyle.Render(r.Timestamp.Format("15:04:05"))
		if i == m.cursor {
			line = selectedStyle.Render(truncate(line, w))
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func (m *Model) renderDetail(list []*core.Request, w, h int) string {
	if len(list) == 0 {
		return dimStyle.Render("no request selected")
	}
	if m.cursor >= len(list) {
		m.cursor = len(list) - 1
	}
	r := list[m.cursor]
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", titleStyle.Render(r.Method), r.Path)
	fmt.Fprintf(&b, "%s\n\n", dimStyle.Render(r.Timestamp.Format("2006-01-02 15:04:05")))

	keys := make([]string, 0, len(r.Headers))
	for k := range r.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\n", dimStyle.Render(k), strings.Join(r.Headers[k], ", "))
	}
	b.WriteString("\n")

	body := string(r.Body)
	if ct := r.Headers.Get("Content-Type"); strings.Contains(ct, "json") && len(r.Body) > 0 {
		var tmp any
		if json.Unmarshal(r.Body, &tmp) == nil {
			if pretty, err := json.MarshalIndent(tmp, "", "  "); err == nil {
				body = string(pretty)
			}
		}
	}
	b.WriteString(truncateMulti(body, w, h-6-len(keys)))
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) <= w {
		return s
	}
	if w < 2 {
		return s[:w]
	}
	return s[:w-1] + "…"
}

func truncateMulti(s string, w, h int) string {
	if h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, l := range lines {
		lines[i] = truncate(l, w)
	}
	return strings.Join(lines, "\n")
}
