# WeazlWrite

![WeazlWrite screenshot](weazlwrite.png)

A sovereign text editor for a paranoid age. WeazlWrite is a private, local-first Markdown writing TUI for vLLM and Ollama servers. Think of it as a quiet terminal desk for drafts, docs, notes, and little technical spells, backed by an encrypted vault tucked under the floorboards.

No web wrappers, no account portals, no telemetry drops, and no browser tabs breeding in the background. Just your files, your vaults, your models, and the blinking cursor.

## Defaults

On first launch, WeazlWrite drops a fresh `config.json` into `~/.config/weazlwrite/` with sensible local defaults:

- `local-vllm`: `http://localhost:8000`
- model: `local-model`
- `local-ollama`: `http://localhost:11434`

Because hardcoding endpoints into a writing tool is how tiny annoyances become permanent roommates, WeazlWrite reads the endpoint and model from the config at runtime.

Encrypted vaults live under `~/.weazlwrite/vault`. Vault notes are stored in SQLite with password-protected vaults and AES-GCM encrypted payloads, but the TUI presents each vault as a standard filesystem tree. Keep plain files on disk, lock private notes in the vault, or bounce a draft between both worlds.

## Run

```sh
go run ./cmd/weazlwrite
go run ./cmd/weazlwrite ./notes/example.md
```

## Grab The Source

```sh
./scripts/install.sh
```

No wizards. No corporate installers. The script handles the chores: it builds `weazlwrite`, tucks it into `~/.weazlwrite/bin`, and adds that directory to your shell `PATH`.

During setup, you will be prompted for your provider type and URL. The script queries the provider for available models, writes `~/.config/weazlwrite/config.json`, and boots straight into the TUI.

Provider URL rules: bare-metal base URLs only.

- vLLM: `https://host:port` or `https://host`, without `/v1`
- Ollama: `http://host:11434`, without `/api`

If you accidentally paste the `/v1` or `/api` suffixes, the installer quietly sanitizes them for you.

Set `WEAZLWRITE_SKIP_LAUNCH=1` to install and configure without triggering the TUI.

## Build From Source

WeazlWrite is a Go app, but it uses SQLite through `go-sqlite3`. Builds require Go 1.25 or newer, CGO, and a working C compiler. It is built to run on solid, reliable standards-based systems like Ubuntu LTS. That C compiler requirement is the one little bit of yak hair you have to shave.

```sh
go build -o weazlwrite ./cmd/weazlwrite
go build -o weazlwrite-setup ./cmd/weazlwrite-setup
```

Useful environment overrides:

- `WEAZLWRITE_CONFIG=/path/to/config.json`
- `WEAZLWRITE_DATA=/path/to/data-dir`

## Keys

- startup vault picker: `up` / `down` choose, `enter` opens, `n` creates a new vault
- `ctrl+e`: edit mode
- `ctrl+r`: rendered preview mode
- `ctrl+o`: show or hide the file tree
- `tab`: move between the file tree and the main writing surface
- `enter`: open a selected file, or fold/unfold a selected folder
- `space`: pick up a file or note; move to a folder; press `space` again to drop it
- `n`: create a new folder from the tree
- `d`: delete the selected file, note, or empty folder
- `r`: rename or move the selected tree item by typing its new path
- `o`: toggle Eyes Only on a vault note
- `i`: import the selected filesystem file or folder into the encrypted vault
- `ctrl+s`: save to the current target
- `ctrl+v`: save to the encrypted vault
- `alt+f`: save to a filesystem path
- `ctrl+f`: find text in the current pane
- `ctrl+g`: jump to a page in the current pane
- `ctrl+p`: ask the local model to insert a Markdown block
- `ctrl+n`: new vault note
- `ctrl+y`: toggle mouse capture off/on for terminal text selection
- `alt+o`: toggle Eyes Only on the current vault note
- `ctrl+k`: open the full command popup
- `?` or `h`: open the full help screen
- `pgup` / `pgdown`: page the focused tree, editor, or render pane
- mouse wheel: scroll the tree or active writing surface
- `ctrl+c`: quit

## Vault And Files

Each vault is an encrypted SQLite database disguised as a note tree. Save something as `projects/specs/api.md`, and WeazlWrite displays it cleanly under `Vault / projects / specs / api.md`.

On startup, WeazlWrite scans `~/.weazlwrite/vault` and throws a vault picker. Pick an existing vault, or press `n` to spin up a new context. New vaults require password confirmation before the database is forged. The selected vault path locks into your config, but the picker stays available on launch so context switching stays cheap.

The left rail splits your brain in two: `Vault` for the encrypted underground, and `Files` for regular surface-level filesystem work. The active note gets a tiny marker so you know exactly where you are without the tree turning into a blinking holiday display. A `*` means the current buffer has unsaved changes.

Big directories are fine. Move with `j` / `k`, the arrow keys, `pgup` / `pgdown`, or the mouse wheel; the tree keeps the selected row in view instead of pretending the world ends at the bottom of the pane.

Tree chores happen right where your cursor is. Press `n` for a new folder, `d` to delete a selected file or empty folder, and `r` to rename or move. Press `space` to pick up a file, navigate, and press `space` again to drop it. Folders fold and unfold with `enter`.

Press `ctrl+v` to save the current buffer into the encrypted vault. Press `alt+f` to save it out to the regular filesystem. Press `ctrl+s` when you simply want to save back to wherever the current note already lives.

To pull existing surface files into the encrypted vault, select a `.md`, `.markdown`, `.txt`, `.pdf`, or `.docx` file and press `i`. Select a folder and press `i` to bulk-import it as a vault root, perfect for absorbing Obsidian vaults that already live on disk. Word and PDF files are aggressively stripped down and converted to pure Markdown before encryption. Image-only PDFs and image-only Word files are rejected because there is no text to harvest.

Long documents get simple navigation help. `pgup` / `pgdown` page through the current edit or render pane, `ctrl+g` jumps to a page number, and `ctrl+f` finds text from your current position.

Copying text is deliberate. Mouse scrolling and terminal drag-selection fight over the same events, so press `ctrl+y` to turn mouse capture off, select/copy text from the editor or renderer with your terminal, then press `ctrl+y` again to restore mouse scrolling.

## Eyes Only Mode

Some notes belong in the vault, not lingering in your system clipboard. Eyes Only mode lets you lock down an encrypted note so it stays readable and editable, without becoming easy prey for an accidental text harvest.

Hit `o` from the tree, or `alt+o` while the vault note is open. Eyes Only files glow high-alert orange in the tree, so you know exactly what you're dealing with before you hit enter.

When an Eyes Only note is live, WeazlWrite hijacks terminal mouse capture and kills the `ctrl+y` selection toggle. You can read the file, grind out edits, and save it back to the encrypted vault, but you absolutely cannot switch into terminal drag-selection mode to copy blocks of text out.

Dropping the shield requires actual intent. Press `o` or `alt+o` and explicitly confirm the warning prompt before the note returns to standard behavior.

This isn't magic DRM, and it won't stop someone with a smartphone from taking a picture of your monitor. It's an anti-foot-gun mechanism for paranoid drafting: encrypted on the metal, visible when unlocked, and deliberately frustrating to casually copy.

That split rail is the entire point: draft in the open when the code belongs in a repo, tuck private notes into the vault when they should not leave the bare metal.

## Markdown And AI

Edit mode is for grinding out text. Render mode is for reading it back without syntax shouting over the prose. WeazlWrite is built on the beautiful Bubble Tea UI framework and uses Glamour so headings, code blocks, and tables keep their terminal-native shape without turning your TUI into a bloated browser.

Need a generated Markdown block? Press `ctrl+p`, describe the spell, and WeazlWrite hits your configured local model for the exact insertable text. While the model grinds, terminal spinners and rotating Weazl-style status phrases keep the screen alive so you know the hardware is working.

## Security

WeazlWrite is strictly local-first and built for paranoia, but it is a TUI app, not a hardware security module. The bcrypt checks and AES-GCM payloads exist to lock out casual snooping and keep your data sovereign. They are not a guarantee that a weak password will survive a dedicated offline attack if someone physically steals your rig.

- Vault databases live locally under `~/.weazlwrite/vault`.
- Vault payloads are encrypted with AES-GCM after unlock.
- API keys belong in your local config, nowhere else.
- Filesystem saves are plain files. Vault saves are encrypted records. Choose accordingly.

## License And Branding

WeazlWrite is released under the MIT License. Fork it, learn from it, ship it.

The `WeazlWrite` name, screenshot, and cyborg-ferret branding are part of this project's identity. If you publish a heavily modified fork, strip the branding and rename it so users know who is actually maintaining the code.
