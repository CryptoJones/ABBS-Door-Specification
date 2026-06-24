<p align="center"><em>Proudly Made in Nebraska. Go Big Red! 🌽 <a href="https://xkcd.com/2347/">https://xkcd.com/2347/</a></em></p>

# ABBS Door Specification

The standard for building **door games** for [AdmiralBBS](https://github.com/CryptoJones/AdmiralBBS) —
a clean-room, security-hardened 90s-style ANSI BBS in Go.

A "door" is an external program the BBS hands a caller off to: a game, a utility,
a chat — anything. This repo is the developer contract. Build to it in **any
language** and your door will run on any AdmiralBBS node.

> ⚠️ **A door is a text-mode terminal program, not a GUI app.** Its only
> interface is a stream of bytes to and from the caller's **VT100/ANSI terminal
> emulator** (over SSH/Telnet). You read keystrokes and write characters and ANSI
> escape codes — that's it. There is no window, no mouse, no framebuffer, no HTTP.
> A Windows `.exe` with a window, a web app, or anything that needs a desktop
> **will not work** — it must be a console program that speaks stdin/stdout (or a
> socket) and renders with ANSI text. See [SPEC.md §4](SPEC.md#4-terminal-conventions).

> **Status:** v1. The normative contract is in **[SPEC.md](SPEC.md)**. Runnable
> starting points are in **[examples/](examples/)**.

## The two door models

| | **Subprocess door** | **Resident door** |
|---|---|---|
| Lifecycle | BBS spawns one process **per player** | One **persistent** server the BBS bridges to |
| Players | single-player, or turn-based shared state via `$DOORSHARE` | real-time **multiplayer**, one shared world |
| I/O | stdin = keystrokes, stdout = screen | raw TCP/Unix-socket byte stream |
| Caller context | `door32.sys` dropfile | none by default — your server prompts |
| Examples | classic LORD-style games, utilities | MajorMUD-style MUDs |
| Reference | the bundled `numguess` demo | [Chrome Circuit Cowboys](https://github.com/CryptoJones/ChromeCircuitCowboys) |

Pick **subprocess** for the simplest path (read a dropfile, talk over
stdin/stdout). Pick **resident** when many callers must share one live world.

## 60-second subprocess door

```sh
#!/bin/sh
# A door is just a program that reads door32.sys and talks over stdin/stdout.
handle=$(sed -n '7p' "${DOORFILE:-door32.sys}" | tr -d '\r')
printf '\033[1;36mWelcome to the door, %s!\033[0m\r\n' "$handle"
printf 'Press Q to quit: '
while IFS= read -r key; do
  case "$key" in [Qq]*) break ;; esac
  printf 'You pressed: %s\r\n> ' "$key"
done
```

Register it from the SysOp control panel (**Content → register door →
subprocess**) with the path to the script. That's a working door. See
[examples/hello-door.sh](examples/hello-door.sh) for a commented version and
[examples/resident-skeleton.go](examples/resident-skeleton.go) for a multiplayer
server skeleton.

## What the BBS guarantees you

- **A clean environment.** Subprocess doors inherit *no* BBS environment — only
  a minimal scrubbed set (`PATH`, `HOME`, `TERM`, `DOORFILE`, optional
  `DOORSHARE`). The BBS master key and every other secret are invisible.
- **A sandbox.** CPU rlimit, wall-clock timeout, and process-group cleanup on
  disconnect. Deployments may further drop privileges / chroot / namespace you.
- **A terminal.** ANSI or plain ASCII (the dropfile tells you which); emit
  `\r\n` line endings.

## What you owe the BBS

- Read input a byte/line at a time and **handle backspace and CR/LF yourself**
  (callers arrive in raw mode).
- **Exit cleanly** when the caller quits or input closes (EOF). Don't wedge.
- Keep your blast radius small: stay in your working directory, don't assume
  network access, and treat all input as hostile.

Full details, field-by-field, in **[SPEC.md](SPEC.md)**.

## Licensing & your doors

This spec and its examples are **MIT-licensed** ([LICENSE](LICENSE)) — use them
however you like, including in **commercial and closed-source** products. MIT
explicitly permits use, modification, distribution, sublicensing, and **selling**.

**Your door game is yours.** Building to this interface does not make your
program a derivative of the spec, and nothing here restricts what you do with a
door you write: sell it, keep it proprietary, license it your own way — your
call. The MIT terms only ask that you keep the notice if you reuse *this repo's*
text or example code.

## Contributing

Improvements to the spec, new example doors, and showcase submissions are
welcome — see **[CONTRIBUTING.md](CONTRIBUTING.md)**. Release history is in
**[CHANGELOG.md](CHANGELOG.md)**.

---
*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
