# Changelog

All notable changes to the **ABBS Door Specification** are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this repo follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

- **PATCH** (`1.0.x`) — fixes, wording/clarifications, and docs/example changes
  that don't change what a conforming door must do.
- **MINOR** (`1.x.0`) — backward-compatible normative additions (new optional
  capabilities a door MAY use; existing doors keep working).
- **MAJOR** (`x.0.0`) — a breaking change to the door contract. A major release
  also bumps the **normative spec version** (`SPEC.md` §7) and lands behind a new
  format identifier rather than silently redefining behavior.

> The **repo release version** (below) and the **normative spec version**
> (`SPEC.md`, currently **v1.1**) are tracked separately: doc and clarification
> releases move the repo version without changing the v1 door contract.

## [Unreleased]

_Nothing yet._

## [1.1.0] - 2026-06-24

Backward-compatible additions derived from the live AdmiralBBS + Chrome Circuit
Cowboys implementations. Normative spec: **v1.1** (a v1 door stays conformant).

### Added
- **Resident-door version handshake** (`SPEC.md` §2.2) — an optional OSC-framed
  `ESC ] ABBS;version=<version> BEL` a door MAY send as its first bytes; the host
  strips it, sanitizes it, and MAY show it (e.g. on the launch line). 1.5s
  handshake timeout; never required.
- **Resident-door input contract** (`SPEC.md` §2.3, normative) — what was
  previously only "copy the reference" is now written down: telnet IAC (`0xFF`)
  skip, non-blocking CR/LF partner-swallow, the `\b \b` backspace echo, the
  `0x20`–`0x7E` printable range, and ignoring NUL.
- **Managed-prompt redraw** (`SPEC.md` §2.4) and **graceful shutdown**
  (`SPEC.md` §2.5) as recommended patterns for real-time multiplayer doors.
- **Host defaults documented**: `TERM` (`ansi`/`dumb`) in §1.2, the
  `<doors-data>/<slug>/node<N>/` + `/shared/` working-dir layout in §1.3, and
  the AdmiralBBS `-door "name|network|address|minlevel"` startup flag + isolation
  flags in §6.

### Changed
- `door32.sys` line 9 (minutes left) clarified as **advisory** — the BBS enforces
  the real time budget, so a door can't extend a session by ignoring it
  (`SPEC.md` §1.4).
- Reference resident door updated from the bundled *Console Cowboy 2026* to
  **Chrome Circuit Cowboys** (now its own repo) across `SPEC.md`, `README.md`,
  and `examples/resident-skeleton.go`, reflecting the AdmiralBBS 2.0 door
  carve-out.

## [1.0.0] - 2026-06-24

Initial public release of the door-game standard for
[AdmiralBBS](https://github.com/CryptoJones/AdmiralBBS). Normative spec: **v1**.

### Added
- **Two door models** — `subprocess` (one process per caller, stdin/stdout) and
  `resident` (a persistent server the BBS bridges to for real-time multiplayer).
- **Subprocess contract** (`SPEC.md` §1): I/O over stdin/stdout, the scrubbed
  environment (`PATH`/`HOME`/`TERM`/`DOORFILE`/optional `DOORSHARE`), per-session
  working directory, the 11-line **`door32.sys`** dropfile format, and the
  sandbox/resource limits (CPU rlimit, wall-clock timeout, process-group kill).
- **Resident contract** (`SPEC.md` §2): the transparent TCP/Unix byte-bridge,
  the server's responsibilities, and a recommended single-goroutine architecture.
- **Saving state** (`SPEC.md` §3): how subprocess doors persist to files
  (`$DOORSHARE`, per-door; concurrency via lockfile + write-then-rename) and how
  resident doors own their store; the BBS persists nothing for doors by design.
- **Terminal conventions** (`SPEC.md` §4): doors are text-mode **VT100/ANSI**
  terminal programs — not GUI/Windows/web apps.
- **Security expectations + Trust model & isolation** (`SPEC.md` §5): the door
  sandbox always protects the BBS key/store and confines a door to its own game
  state; hard door-to-door and door-to-host isolation requires deploy-time
  per-door UID / chroot / namespaces.
- **Registration data model** (`SPEC.md` §6) and **versioning policy**
  (`SPEC.md` §7).
- **Runnable examples**: a commented subprocess door (`examples/hello-door.sh`)
  and a multiplayer resident server skeleton (`examples/resident-skeleton.go`).
- **`CONTRIBUTING.md`** — how to propose spec changes and example standards.
- **MIT License** — commercial and closed-source doors are explicitly permitted.

[Unreleased]: https://github.com/CryptoJones/ABBS-Door-Specification/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/CryptoJones/ABBS-Door-Specification/releases/tag/v1.1.0
[1.0.0]: https://github.com/CryptoJones/ABBS-Door-Specification/releases/tag/v1.0.0
