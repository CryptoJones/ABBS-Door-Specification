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
> (`SPEC.md`, currently **v1**) are tracked separately: doc and clarification
> releases move the repo version without changing the v1 door contract.

## [Unreleased]

_Nothing yet._

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

[Unreleased]: https://github.com/CryptoJones/ABBS-Door-Specification/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/CryptoJones/ABBS-Door-Specification/releases/tag/v1.0.0
