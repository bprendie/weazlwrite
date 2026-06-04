# Security Policy

WeazlWrite is a local-first Markdown TUI for personal writing and encrypted vault notes. Its vault protections are intended to reduce casual snooping and accidental exposure on a local machine; they are not a substitute for full-disk encryption, strong operating-system account security, or a hardware security module.

## Supported Use

- Keep vault passwords strong and unique.
- Keep `config.json` local to the machine and protect API keys stored there.
- Use `vault.auto_lock_minutes` to control the inactivity timeout. The default is 15 minutes. Set it to `0` only if you explicitly want to disable session auto-lock.
- Treat filesystem saves as plaintext. Only vault saves are encrypted.

## Current Mitigations

- Vault note payloads are encrypted with AES-GCM after unlock.
- Vault password checks use bcrypt.
- Failed unlock attempts use increasing in-memory backoff delays.
- Unlocked sessions auto-lock after inactivity and clear visible document content.
- Config files are written with `0600` permissions on supported platforms.

## Reporting Issues

Report security issues privately to the project maintainer before public disclosure. Include the affected version or commit, operating system, reproduction steps, and whether an existing vault is required to reproduce the issue.
