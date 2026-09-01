# Changelog

## Unreleased

- Remap writing keys so the editor owns motion while focused: `alt+v` saves to the vault, `alt+d` saves to disk, `alt+i` inserts AI, `alt+n` creates a vault note, `f1`/`?` open help. `ctrl+v` pastes, `ctrl+e` is end of line in the editor, and tree `ctrl+n` still creates a folder.
- Upgrade Go runtime target to 1.25.10 or newer for security fixes.
- Update `golang.org/x/net` to `v0.53.0` and related `golang.org/x/*` modules.
- Add vault password attempt rate limiting with increasing in-memory backoff delays.
- Add configurable vault session auto-lock via `vault.auto_lock_minutes`.
- Save dirty work before auto-lock when possible, then clear visible vault document state.
