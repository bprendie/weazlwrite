# weazlwrite

Terminal Markdown writing for the Weazl app suite.

`weazlwrite` is a dark, cyberpunk-flavored Markdown editor with:

- password-gated local SQLite vault storage
- encrypted vault notes using bcrypt-derived unlock flow and AES-GCM content storage
- live Markdown preview rendered with Glamour
- AI block insertion through local Ollama or vLLM providers
- local directory access for opening and saving `.md`, `.markdown`, and `.txt` files
- direct terminal opening with `weazlwrite ./somefile.md`

## Build

```sh
go build -o weazlwrite ./cmd/weazlwrite
```

## Run

```sh
./weazlwrite
./weazlwrite ./notes/example.md
```

Configuration is created at `~/.config/weazlwrite/config.json` by default. Data is stored under `~/.local/share/weazlwrite`.

Useful environment overrides:

- `WEAZLWRITE_CONFIG=/path/to/config.json`
- `WEAZLWRITE_DATA=/path/to/data-dir`

## Keys

- `ctrl+s` save
- `ctrl+i` prompt the local AI model to insert a Markdown block at the cursor
- `ctrl+n` new vault note
- `ctrl+o` focus file tree
- `ctrl+e` focus editor
- `ctrl+p` focus preview
- `pgup` / `pgdn` scroll preview
- `ctrl+c` quit
