# Changelog

## Unreleased

- Page the editor by wrapped on-screen rows (pgup/pgdown, status page count, ctrl+g). Home/End stay start/end of the Markdown paragraph.
- Uncap the editor so Enter works past 99 lines, wrap follows the pane, the extra prompt gutter is gone, line numbers stay a stable width, and tabs in a file are not rewritten on open+save.
- Remap writing keys so the editor owns motion while focused: `alt+v` saves to the vault, `alt+d` saves to disk, `alt+i` inserts AI, `alt+n` creates a vault note, `f1`/`?` open help. `ctrl+v` pastes, `ctrl+e` is end of line in the editor, and tree `ctrl+n` still creates a folder.
- Upgrade Go runtime target to 1.25.10 or newer for security fixes.
- Update `golang.org/x/net` to `v0.53.0` and related `golang.org/x/*` modules.
- Add vault password attempt rate limiting with increasing in-memory backoff delays.
- Add configurable vault session auto-lock via `vault.auto_lock_minutes`.
- Save dirty work before auto-lock when possible, then clear visible vault document state.
