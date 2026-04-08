# 🐨 kowari

> Catch webhooks locally. No cloud. No leaks. A terminal webhook receiver that never phones home.

**[→ kowari.dev landing page](https://iamkorun.github.io/kowari)**

[![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-yellow.svg)](LICENSE)
[![Stars](https://img.shields.io/github/stars/iamkorun/kowari?style=social)](https://github.com/iamkorun/kowari)

## The Problem

You're building a Stripe / GitHub / Slack integration. You need to see what the webhook actually sends — and replay it into your local server twenty times while you fix a bug. Tunnel services want an account. Browser UIs hide the JSON. Most tools need the internet.

## The Solution

`kowari` runs locally, captures every request hitting `:8080`, shows them in a split-pane TUI, and replays any of them to your dev server with a keystroke. No account, no cloud, no tracking.

## Quick Start

```bash
go install github.com/iamkorun/kowari@latest
kowari --port 8080 --target http://localhost:3000
```

## Installation

```bash
# Go
go install github.com/iamkorun/kowari@latest

# From source
git clone https://github.com/iamkorun/kowari
cd kowari && go build -o kowari .
```

## Usage

```bash
# Listen on :8080, replay to a local service
kowari --target http://localhost:3000

# Persist every captured request as JSONL
kowari --save hooks.jsonl

# Headless mode (no TUI) for CI/tests
kowari --headless --save hooks.jsonl
```

### Keybindings

| Key | Action |
|-----|--------|
| ↑ / ↓ (or `k`/`j`) | Navigate requests |
| `r` | Replay selected request to `--target` |
| `c` | Clear all captured requests |
| `q` | Quit |

## Features

- 🪄 Zero config — one binary, one flag
- 📬 Captures every method, every path, every header
- 🔁 One-keystroke replay to any target URL
- 💾 Optional JSONL persistence (`--save`)
- 🎨 Split-pane TUI with pretty-printed JSON bodies
- 🔌 Offline-first — no accounts, no cloud, no tracking
- 📦 Single static binary (Go)

## Contributing

Pull requests welcome. Run `go test ./...` before submitting.

## License

MIT

---

## Star History

<a href="https://star-history.com/#iamkorun/kowari&Date">
  <img src="https://api.star-history.com/svg?repos=iamkorun/kowari&type=Date" alt="Star History Chart" width="600">
</a>

---

<p align="center">
  <a href="https://buymeacoffee.com/iamkorun"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me a Coffee" width="200"></a>
</p>
