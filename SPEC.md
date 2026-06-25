# ABBS Door Specification — v1.2

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
| `TERM` | the caller's terminal type: `ansi` for an ANSI-capable caller, `dumb` otherwise (the host defaults it to `ansi` when none is given) |
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

On the AdmiralBBS reference host the layout is concretely
`<doors-data>/<slug>/node<N>/` for each node's working directory (`$HOME`, where
`door32.sys` lives) and `<doors-data>/<slug>/shared/` for `$DOORSHARE`, where
`<slug>` is the door name lowercased to `[a-z0-9-]`. A door launched with no
working directory configured runs in a throwaway temp jail that is deleted on
exit (no persistent state). Don't hardcode these paths — read `$HOME`,
`$DOORFILE`, and `$DOORSHARE` — but this is the shape to expect when testing.

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
| 9 | Minutes left this session | integer; **advisory** — display it if you like, but the BBS enforces the caller's real time budget itself, so a door can't extend a session by ignoring or rewriting this value |
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
  then relays bytes verbatim in both directions: caller→server and
  server→caller. Apart from the **one optional version handshake** your server
  may send first (§2.2), there is **no framing or added protocol** — the relay is
  a transparent byte pipe.
- Your server **MUST** drive its own terminal experience over that stream:
  prompt for whatever identity/character it needs, do its own raw line editing
  and echo (§2.3), and emit ANSI if it wants color.
- Your server **MUST** tolerate many simultaneous connections, connections that
  vanish without warning (caller hangup), and hostile input.
- The BBS passes **no dropfile and no environment** to a resident door — it only
  relays the stream. If you need the caller's BBS handle, prompt for a name (as
  classic MUDs do) or design your own login.
- Your server owns its own persistence, world state, and lifecycle. It starts
  and stops independently of the BBS.

### 2.2 Version handshake (optional)

A resident door **MAY**, as the **very first bytes** it writes on an accepted
connection — before any prompt or game output — advertise its version with a
single OSC-framed string:

```
ESC ] ABBS;version=<version> BEL
```

that is, the bytes `0x1B 0x5D` (`ESC ]`), the literal ASCII `ABBS;version=`, your
version string, then `0x07` (`BEL`). For example, version 1.0.0 is the bytes:

```
\x1b]ABBS;version=1.0.0\x07
```

Host behavior (AdmiralBBS reference):

- The host reads the sentinel, **strips it from the stream** (the caller never
  sees it), and **MAY** display the version — e.g. on the launch line:
  `Launching Chrome Circuit Cowboys v1.0.0 (node 1)...`.
- The host waits at most a short **handshake timeout** (1.5s) for it. A door that
  sends nothing, or whose first bytes are not this exact sentinel, is treated as
  having no handshake and **all** its bytes are relayed verbatim. Sending it is
  **optional and never required**; the bridge works identically without it.
- The host **sanitizes** the version before display — keeping only the characters
  `[A-Za-z0-9.+-]`, truncated to 32 — so the handshake can never inject control
  sequences into a caller's terminal. Keep your version string within that set.

OSC framing is deliberate: a terminal that receives the sentinel directly (a door
reached without a host) silently swallows it, so it never garbles a raw session.

The handshake payload is a `;`-separated list of `key=value` fields. `version` is
one; a door **MAY** also advertise capabilities it wants the host to act on:

```
ESC ] ABBS;version=<version>;caps=<cap,cap,...> BEL
```

Unknown keys/caps are ignored. The only capability defined today is `handle`
(§2.2a).

### 2.2a Host handle push (capability: `handle`)

A door that advertises **`caps=handle`** in its handshake (§2.2) is telling the
host it wants the caller's handle. The host responds by writing, as the next
bytes back **to the door**, a reciprocal OSC sentinel:

```
ESC ] ABBS;handle=<handle> BEL
```

The door reads and strips it (a short, non-blocking peek), and uses the handle
however it likes — e.g. to default its own name prompt (`Handle [name] (Enter to
use):`). Notes:

- The host sends this **only** to doors that advertised the `handle` capability,
  so it never injects bytes a door isn't expecting.
- The host **sanitizes** the handle to `[A-Za-z0-9_.-]` (≤24 chars).
- A door that asked for it but receives nothing within a short window (a host
  that doesn't support the push) **MUST** fall back to prompting normally.

### 2.3 Input handling (normative)

The relay is raw — there is no line discipline, and clients differ (SSH raw mode;
assorted telnet clients). A resident door **MUST** handle input byte by byte.
Specifically, a door **MUST**:

- **Telnet IAC:** on byte `0xFF` (telnet IAC), consume it **and the following
  byte** and ignore both. The host does **not** strip telnet negotiation for you;
  a door that treats `0xFF` as input will corrupt on real telnet clients.
- **Line submission:** treat CR (`0x0D`) **or** LF (`0x0A`) as "line entered."
  Clients send lone CR, lone LF, CRLF, or LFCR. To collapse a pair without
  hanging, peek **only if a byte is already buffered** (a non-blocking check) and
  swallow it **only** when it is the CR↔LF partner of the byte you just read. A
  door **MUST NOT** block waiting for a partner that may never come — a lone CR
  from an interactive keystroke has to submit immediately. Echo a newline as
  `\r\n`.
- **Editing:** on backspace/DEL (`0x08` or `0x7F`), remove the last buffered
  input byte (if any) and echo the 3-byte sequence `\b \b` (backspace, space,
  backspace) to erase the glyph on screen.
- **Printable range:** buffer and echo only bytes `0x20`–`0x7E`. Silently
  **ignore** NUL (`0x00`) and any other control byte you don't specifically
  handle.

### 2.4 Keeping the prompt readable under async output (recommended)

A real-time multiplayer door emits output the caller didn't trigger — combat
ticks, chat, other players' actions — while the caller is mid-type. A door
**SHOULD** redraw rather than clobber the input line. The pattern the reference
door uses:

- Keep, per connection, the caller's **in-progress (un-submitted) input** and the
  **current status prompt**, guarded by a mutex so output can't interleave with
  input echo.
- To print async content, write `\r\x1b[K` (CR + ESC`[K` — return to column 0 and
  erase the line), then the content, then **redraw** the prompt followed by the
  buffered input. The caller sees the new line appear *above* an intact,
  still-editable prompt.
- Re-show the prompt on state changes (e.g. HP) and, on each world tick, **only
  for connections that received output that tick**, so idle callers don't get a
  repeating prompt.

Skipping this isn't a protocol violation, but multiplayer output will visibly
scramble whatever the caller is typing.

### 2.5 Graceful shutdown (recommended)

A resident door owns its own lifecycle. On `SIGINT`/`SIGTERM` (e.g. a
`systemctl restart`) a door **SHOULD** flush/persist connected players' state
before exiting, with a **bounded** wait so a stuck save can't hang shutdown
forever (the reference door waits up to 5s, then exits). The BBS persists nothing
for you (§3.2).

### 2.6 Recommended architecture

Serialize all shared-world mutation (a single goroutine/thread consuming a
command queue plus a timer "tick") so the core logic needs no locks and stays
deterministic and testable; keep network I/O at the edges. The reference
implementation, **Chrome Circuit Cowboys** (its own repo,
<https://github.com/CryptoJones/ChromeCircuitCowboys>), follows exactly this
shape and is a good model to copy.

### 2.7 Release-install convention (optional)

A host MAY let an operator install a resident door by pointing it at a forge
**release URL** — the host downloads the binary, runs it under supervision, and
registers the bridge automatically (AdmiralBBS does this from the SysOp panel).
To be installable this way, a door **MUST**:

- **Accept `-addr host:port`** and listen on exactly that address. The host picks
  a free localhost port and launches the door as `<binary> -addr 127.0.0.1:<port>`
  with the working directory set to a per-door data dir (persist relative to the
  cwd, or accept your own flags with sane defaults). This `-addr` flag is the
  only launch contract the host relies on.
- **Publish a binary asset per platform** on the release, named with OS and arch
  tokens so the host can pick the one matching its **own** machine — **no OS is
  second-class**. Recommended tokens (case-insensitive, with common aliases):
  OS `linux` / `windows` (`win`) / `darwin` (`macos`); arch `amd64` (`x86_64`) /
  `arm64` (`aarch64`). Examples: `mydoor-linux-amd64`,
  `mydoor-windows-amd64.exe`, `mydoor-darwin-arm64`.

The release JSON must expose `tag_name` and `assets[].browser_download_url` — the
shape GitHub, Codeberg, and Forgejo all return (the same forge-agnostic shape the
update check uses). A door that doesn't publish per-platform assets, or doesn't
take `-addr`, can still be installed the manual way (operator runs it and
registers the bridge by hand); this convention only enables the one-click path.

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
logout/disconnect and/or periodically). The reference resident door, **Chrome
Circuit Cowboys**, uses its **own SQLite database**, separate from the BBS
database, and saves a character when they disconnect (and flushes everyone on
shutdown, §2.5).

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

On the AdmiralBBS reference host, an operator wires a **resident** door at
startup with a repeatable flag encoding that data model as
`name|network|address|minlevel`:

```
admiralbbs -door "Chrome Circuit Cowboys|tcp|127.0.0.1:4000|0"
```

`minlevel` is optional (defaults to 0). The flag is repeatable to register
several doors, and registration is idempotent on `name`. Subprocess-door
sandbox hardening (§1.5) is configured host-wide, not per door, via
`-door-uid`, `-door-gid`, `-door-chroot`, `-door-no-network`, and
`-door-isolate`. (These flags are AdmiralBBS-specific; other hosts may register
the same data model however they like.)

## 7. Versioning

This is **v1.2**. Each minor is backward compatible: v1.1 added the optional
resident-door version handshake (§2.2) and wrote down the resident-door input
contract (§2.3–§2.5); v1.2 adds the optional release-install convention (§2.7).
No existing field changed meaning, so a v1 door remains conformant. The
`door32.sys` dropfile is still fixed at 11 lines; new fields, if any, will be
appended in a later version behind a new format identifier.

---
*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
