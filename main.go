// Command kowari runs an offline-first TUI webhook receiver.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iamkorun/kowari/core"
	"github.com/iamkorun/kowari/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	port := flag.Int("port", 8080, "port for the webhook receiver")
	target := flag.String("target", "", "URL to replay captured requests to")
	save := flag.String("save", "", "append captured requests to this JSONL file")
	headless := flag.Bool("headless", false, "run without the TUI (useful for CI/tests)")
	flag.Parse()

	store := core.NewStore(*save)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           store.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "server error:", err)
			os.Exit(1)
		}
	}()

	if *headless {
		fmt.Printf("kowari listening on :%d (headless)\n", *port)
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		return
	}

	m := tui.New(store, *target, *port)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.SetProgram(p)
	store.OnAdd(func(*core.Request) { m.Notify() })

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
