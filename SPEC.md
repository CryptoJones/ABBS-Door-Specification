# ABBS Door Specification — v1

This is the normative contract for door games on AdmiralBBS. The key words
**MUST**, **SHOULD**, and **MAY** are used in the usual sense.

There are two door **kinds**: `subprocess` and `resident`. A door declares its
kind at registration time (see §6).

---

## 1. Subprocess doors

The BBS launches a fresh process **per caller** and wires the caller's terminal
to the process's standard streams.

### 1.1 Process & I/O contract

- The door is executed as `/bin/sh -c 'ulimit -t <cpu>; exec "$@"' sh <command> [args...]`.
  Your `<command>` is run with its working directory set to a **per-session
  working directory** (see §1.3).
- **stdin** delivers the caller's keystrokes as a raw byte stream.
- **stdout** is written verbatim to the caller's terminal.
- **stderr** is discarded.
- The door **MUST** treat stdin as raw: no line discipline is guaranteed. The
  door **MUST** handle CR (`\r`), LF (`\n`), and backspace (`0x08`/`0x7f`)
  itself, and **SHOULD** echo typed characters if it wants the caller to see
  them.
- The door **SHOULD** emit `\r\n` for newlines.
- The door **MUST** exit when stdin reaches EOF (the caller disconnected or the
  BBS is tearing the session down). It **MUST NOT** block forever.

### 1.2 Environment

The process environment is **scrubbed** — nothing from the BBS daemon is
inherited. Exactly these variables are provided:

| Variable | Meaning |
|---|---|
| `PATH` | `/usr/bin:/bin` |
| `HOME` | the per-session working directory |
| `TERM` | the caller's terminal type (e.g. `ansi`) |
| `DOORFILE` | absolute path to this session's `door32.sys` dropfile |
| `DOORSHARE` | *(optional)* absolute path to a per-door directory shared by all players — present only for multiplayer doors |

A door **MUST NOT** rely on any other inherited variable.

### 1.3 Working directory and shared state

- Each session runs in a **working directory** that contains this session's
  `door32.sys`. For multiplayer-aware doors this directory is unique per node,
  so concurrent callers never clobber each other's dropfile.
- Turn-based / asynchronous multiplayer doors share state through the directory
  named by **`$DOORSHARE`** (same directory for every concurrent player of that
  door). Doors **MUST** assume concurrent access and lock/serialize their own
  files there (e.g. via lockfiles). Real-time multiplayer should use a
  *resident* door instead (§2).

### 1.4 The `door32.sys` dropfile

The BBS writes a standard **door32.sys** — 11 lines, CRLF-terminated — into the
working directory before launch. `$DOORFILE` points at it. The lines, in order:

| Line | Field | Notes |
|---|---|---|
| 1 | Comm type | `2` (telnet-style handle) |
| 2 | Comm/socket handle | `0` (I/O is via stdin/stdout, not a socket) |
| 3 | Baud rate | `115200` (nominal) |
| 4 | BBS name | |
| 5 | User record position | `1` |
| 6 | User real name | falls back to the handle if unset |
| 7 | **User handle / alias** | the caller's BBS handle |
| 8 | Access level | integer |
| 9 | Minutes left this session | integer; honor it |
| 10 | Terminal emulation | `0` = ASCII, `1` = ANSI |
| 11 | **Node number** | unique per concurrent session |

Doors **SHOULD** read at least the handle (line 7), emulation (line 10), and
node (line 11). Doors **SHOULD NOT** assume any field beyond line 11 exists.

### 1.5 Sandbox & resource limits

The BBS enforces, and a door **MUST** tolerate:

- a **CPU-seconds rlimit** (default 120s) applied via `ulimit -t`;
- a **wall-clock timeout** (default 15 minutes) after which the door's entire
  process group is `SIGKILL`'d;
- **process-group teardown** on caller disconnect — the door and any children
  it spawned are killed together.

Deployments **MAY** additionally run doors under a dedicated unprivileged UID, a
`chroot`, and/or fresh Linux namespaces (no network, isolated mount/PID/IPC). A
door **MUST NOT** assume it has network access, write access outside its working
and share directories, or any ambient privilege.

---

## 2. Resident doors

A resident door is a **long-running server** you operate. It listens on a TCP or
Unix socket. When a caller opens the door, the BBS **bridges** the caller's
session to your server by relaying raw bytes in both directions until either
side closes. Every caller is bridged to the **same** server process, so they
share one live world — this is how real-time multiplayer works.

### 2.1 Connection contract

- The BBS dials your `address` (see §6) with a connect timeout (default 10s) and
  relays bytes verbatim: caller→server and server→caller. There is **no framing,
  handshake, or added protocol** — it is a transparent byte pipe.
- Your server **MUST** drive its own terminal experience over that stream:
  prompt for whatever identity/character it needs, do its own line editing
  (raw input — handle CR/LF and backspace), and emit ANSI if it wants color.
- Your server **MUST** tolerate many simultaneous connections, connections that
  vanish without warning (caller hangup), and hostile input.
- The BBS passes **no dropfile and no environment** to a resident door — it only
  relays the stream. If you need the caller's BBS handle, prompt for a name (as
  classic MUDs do) or design your own login.
- Your server owns its own persistence, world state, and lifecycle. It starts
  and stops independently of the BBS.

### 2.2 Recommended architecture

Serialize all shared-world mutation (a single goroutine/thread consuming a
command queue plus a timer "tick") so the core logic needs no locks and stays
deterministic and testable; keep network I/O at the edges. The reference
implementation, **Console Cowboy 2026** (`src/game/cowboy` + `src/cmd/cowboy` in
the AdmiralBBS repo), follows exactly this shape and is a good model to copy.

---

## 3. Saving state

**Saving state is the door's job, not the BBS's.** The BBS deliberately stores
nothing about a door's progress — it only provides the directories, the
dropfile, the sandbox, and (for resident doors) the byte pipe. Keeping door data
outside the BBS's own encrypted store is what guarantees a door can never reach
the BBS master key or another caller's secrets. How you persist depends on the
door kind.

### 3.1 Subprocess doors — files

A subprocess door persists to **files**, in one of two BBS-provided locations:

- **`$DOORSHARE`** — a directory **scoped to this one door** and shared by *all
  concurrent players of that door* (the same path for every node of this door).
  It is **not** shared with other doors — each door has its own share directory,
  so one game cannot reach another game's saves through `$DOORSHARE`. This is
  where a door's cross-player, persistent state lives: player records, high-score
  tables, world/map data. `$DOORSHARE` is present only for doors the operator has
  configured as multiplayer/shared. (See §5.1 for the isolation guarantees and
  their limits.)
- **The per-session working directory** (`$HOME`, where `door32.sys` lives) —
  unique per node. Use it for scratch/temp files. It is **not** shared, so it is
  the wrong place for persistent cross-player state.

A door identifies *which* player it's saving by reading the caller's handle from
the dropfile (`door32.sys` line 7), then loading/updating that player's record.

Because callers run **concurrently**, a door **MUST** serialize its own access
to `$DOORSHARE`:

- guard shared files with a lockfile (e.g. create-exclusive a `.lock`, or
  `flock(2)`), and keep the critical section short;
- make updates crash-safe by writing to a temp file in the same directory and
  `rename(2)`-ing it over the target (atomic on POSIX);
- never assume you are the only running instance.

The BBS does **no** locking for you. A door that ignores this will corrupt its
save files under concurrent play. (If your design needs continuous real-time
shared state rather than turn-based file swaps, use a *resident* door instead —
§3.2.)

### 3.2 Resident doors — your own store

A resident door is a long-lived server (§2), so it simply **keeps its own
state** — typically the world in memory plus a database or files it owns
entirely. The BBS only relays bytes; it passes no dropfile and persists nothing
for you. Persist on a cadence that fits your game (e.g. save each character on
logout/disconnect and/or periodically). The reference resident door, **Console
Cowboy 2026**, uses its **own SQLite database**, separate from the BBS database,
and saves a character when they disconnect.

Resident persistence has no cross-process locking problem (one server owns all
the state), but you **MUST** still serialize access *within* your process — the
recommended single-goroutine/command-queue architecture (§2.2) does this for
free.

---

## 4. Terminal conventions

**A door is a text-mode terminal program.** The caller reaches it through a
**VT100/ANSI terminal emulator** over SSH or Telnet. The door's *entire* output
is a stream of bytes — printable text plus ANSI/VT100 escape sequences — rendered
by that terminal; its entire input is the bytes the caller types. A door
therefore **MUST**:

- produce only a byte stream meant for a VT100/ANSI terminal (no GUI windows, no
  mouse, no graphics/framebuffer, no HTML — there is no display surface but the
  terminal);
- be a **console/headless** program: a subprocess door talks over stdin/stdout
  (§1), a resident door over a socket (§2). A program that requires a windowing
  system or desktop (e.g. a Windows GUI `.exe`, an Electron/web app) is **not a
  valid door** and will not run, even if "dropped in".

Cursor movement, color, and clears are done with **ANSI/VT100 escape sequences**
(`\x1b[...`), not any native UI toolkit.

- Line ending: `\r\n`.
- Color: ANSI SGR escape sequences (`\x1b[...m`). Check the caller's emulation
  (dropfile line 10 for subprocess doors) and fall back to plain ASCII when it
  is `0`. Resident doors that cannot detect emulation **SHOULD** offer a
  color on/off choice or keep color modest.
- Input: assume **raw** mode. Do not assume the caller's client echoes or
  buffers lines for you.

## 5. Security expectations

Doors run on someone else's BBS. A door **MUST**:

- treat every byte of input as hostile (no buffer overflows, no command
  injection, bound all reads);
- stay within its working/share directories and its own socket;
- never attempt to read BBS secrets, escape its sandbox, or scan the host;
- exit promptly and cleanly on EOF/disconnect.

Dual-use is fine (a door is arbitrary code by nature); malicious behavior toward
the host or other callers is out of spec and will get a door delisted.

### 5.1 Trust model & isolation

A door is **arbitrary code that a SysOp chooses to install** — running one is a
trust decision, exactly like installing any server software. Callers never
install or upload doors; they only send keystrokes. So "a player smuggled in a
fake game" is not a thing — but "a SysOp installed a malicious door" is, and the
sandbox exists to bound that door's blast radius.

What the sandbox guarantees, **always**:

- A door cannot read the **BBS's own data or master key**. The BBS store is
  encrypted; the key lives only in the daemon's memory and is never placed in a
  door's environment, dropfile, or working files. The door's environment is
  scrubbed (§1.2).
- A door is **authoritative over its own game state and nothing else.** A
  malicious or buggy door can cheat or corrupt *its own* saves — including its
  own "character sheets" — but that is contained to that game. It is not a path
  to the BBS or to other callers' accounts.
- Resource limits (CPU, wall-clock, process-group kill) and, for resident doors,
  the fact that the BBS only relays bytes, apply regardless.

What requires **deploy-time hardening** (honest limitation):

- *Hard* isolation **between doors** and **from the host** — so a hostile door
  truly cannot read a sibling door's files or touch the host — depends on
  running doors under a **dedicated unprivileged UID (ideally one per door),
  `chroot`, and/or Linux namespaces** (§1.5). In the minimal default where every
  door runs as the same UID, the separation between doors is directory
  convention plus the scrubbed environment, so a hostile door sharing that UID
  could read another door's files on disk.

**Operator guidance:** treat door code like any third-party server software —
vet it, and run untrusted or community doors under per-door UID / chroot /
namespace isolation. The BBS's encrypted store is protected either way; the
hardening protects doors from *each other* and the host from a bad door.

## 6. Registration (data model)

A SysOp registers a door from the control panel (**Content → register door**),
or an operator wires one at startup. Each door record carries:

| Field | Subprocess | Resident |
|---|---|---|
| `name` | display name | display name |
| `kind` | `subprocess` | `resident` |
| `command` | path to the executable | *(unused)* |
| `dropfile format` | `door32.sys` | *(unused)* |
| `net_type` | *(unused)* | `tcp` or `unix` |
| `address` | *(unused)* | host:port (tcp) or socket path (unix) |
| `min_access_level` | minimum caller access level | same |

Doors are gated by `min_access_level`: a caller below it never sees the door.

## 7. Versioning

This is **v1**. Additions will be backward compatible where possible; the
dropfile is fixed at 11 lines for v1 and new fields, if any, will be appended in
a later version behind a new format identifier.

---
*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
