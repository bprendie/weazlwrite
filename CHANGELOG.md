# Changelog

## v0.1.0 — 2026-09-05

First tagged release. A terminal writing desk with an encrypted vault and considerably fewer reasons to swear at the editor.

- Quiet rolling vault saves after 750 ms of quiet or five seconds of continuous typing. Ctrl+S flushes immediately; failures keep the draft and offer retry.
- New vault drafts get readable filenames from up to six opening words, with Markdown formatting removed. First Ctrl+S suggests a name to accept or edit; confirmation survives reopening. Duplicate suggestions receive numeric suffixes, and explicit naming cannot overwrite another note.
- Traditional editor controls: Tab/Shift+Tab indentation, Markdown list/number/checkbox continuation, blockquotes, and fenced-code indentation. Paste preserves the supplied text; assisted edits undo together.
- Persistent keyboard/mouse selection, Ctrl+A, Ctrl+C/X/V, and Unicode-aware character movement. Ctrl+Z undoes; Ctrl+Y or Alt+Z redoes. Alt+M toggles mouse capture. Eyes Only blocks copy/cut.
- Inline find/replace with highlighted matches, forward/backward navigation, single-match replacement, and Esc restoring the original search position.
- Adjustable writing width and optional typewriter scrolling, with saved preferences. Shared wrapping and cached row positions improve long-document typing and align mouse placement with the visible text.
- Esc clears selection or closes a popup before moving to the tree. Vault writes finish before document replacement, quit, or auto-lock; unsaved disk drafts offer save/discard/cancel. Ctrl+Q quits.
- Alt+V saves to the vault, Alt+S saves to disk, Alt+I inserts AI output, and Alt+N creates a vault draft. F1 opens command help.
- Local vLLM/Ollama integration, encrypted SQLite vaults, document import, and filesystem editing.
- Vault password attempt rate limiting and configurable session auto-lock. Go runtime target 1.25.10 and updated networking/crypto dependencies.
- Code split by responsibility to stay below the 300-line ceiling, checked by `scripts/check-loc.sh`.

Existing vault notes retain their filenames. The vault gains an additive `auto_named` column to remember unconfirmed draft names; document payload encryption is unchanged.

Validation: Go test suite, TUI/storage race checks, source line ceiling, and installed-app smoke tests passed. Desktop clipboard integration was tested with isolated utilities; the actual desktop clipboard service remains unverified.
