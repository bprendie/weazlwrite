# WeazlWrite

WeazlWrite is a private, local-first Markdown writing TUI for the Weazl app suite. It gives you a quiet terminal workspace, a password-protected SQLite vault, filesystem notes when you want them, Glamour-rendered previews, and a local AI helper for dropping clean Markdown blocks into the draft.

## Defaults

On first launch, WeazlWrite writes `~/.config/weazlwrite/config.json` with local model defaults:

- `local-vllm`: `http://localhost:8000`
- model: `local-model`
- `local-ollama`: `http://localhost:11434`

The encrypted vault lives under `~/.weazlwrite/vault`. Notes inside the vault are stored in SQLite with password-protected access and AES-GCM encrypted payloads, but they are shown in the TUI as a filesystem-style tree.

## Install

```sh
./scripts/install.sh
```

The installer builds `weazlwrite`, tucks it into `~/.weazlwrite/bin`, adds that directory to your shell `PATH`, prompts for a local vLLM or Ollama provider, and launches the app.

Set `WEAZLWRITE_SKIP_LAUNCH=1` to install and configure without starting the TUI.

## Run

```sh
./weazlwrite
./weazlwrite ./notes/example.md
```

## Build From Source

WeazlWrite is a Go app, but it uses SQLite through `go-sqlite3`, so builds need Go 1.25 or newer, CGO, and a working C compiler.

```sh
go build -o weazlwrite ./cmd/weazlwrite
go build -o weazlwrite-setup ./cmd/weazlwrite-setup
```

Useful environment overrides:

- `WEAZLWRITE_CONFIG=/path/to/config.json`
- `WEAZLWRITE_DATA=/path/to/data-dir`

## Keys

- `ctrl+e`: edit mode
- `ctrl+r`: rendered preview mode
- `ctrl+o`: show or hide the file tree
- `tab`: move between the file tree and the main writing surface
- `ctrl+s`: save to the current target
- `ctrl+v`: save to the encrypted vault
- `ctrl+f`: save to a filesystem path
- `ctrl+p`: ask the local model to insert a Markdown block
- `ctrl+n`: new vault note
- mouse wheel: scroll the active surface
- `ctrl+c`: quit
