# Contributing to the ABBS Door Specification

Thanks for helping improve the door standard for
[AdmiralBBS](https://github.com/CryptoJones/AdmiralBBS)! This repo is the
**spec** (`SPEC.md`), its orientation (`README.md`), and **runnable examples**
(`examples/`). It is *not* the BBS engine — engine changes belong in the
AdmiralBBS repo.

By contributing, you agree your contribution is licensed under this repo's
[MIT License](LICENSE).

## What contributions are welcome

- **Clarifications & fixes** to `SPEC.md` / `README.md` — wording, ambiguities,
  typos, broken links.
- **Examples** — new minimal, well-commented door starters in any language
  (a Python subprocess door, a Rust resident server, etc.). Small and
  educational beats large and clever.
- **Compatibility notes** — terminal/client quirks, gotchas, porting tips.
- **Showcase** — built a door? Open an issue to have it listed (see below).

## What goes elsewhere

- Changes to how AdmiralBBS *launches* or *bridges* doors → the
  [AdmiralBBS](https://github.com/CryptoJones/AdmiralBBS) repo. If the engine
  behavior changes, the spec PR here should land alongside (or just after) it so
  the two never disagree.
- Your own door game's source — that's **your** project. You never need to
  contribute it here; the spec puts no claim on doors built to it (sell it, keep
  it closed-source — see the README "Licensing & your doors").

## How to propose a change

1. **Open an issue first** for anything beyond a typo — especially **normative**
   changes (anything that alters what a door MUST/SHOULD do). Describe the
   problem and the proposed wording so it can be discussed before code.
2. **Branch and PR.** Fork (or branch), make the change, open a pull request
   against `main`. Keep PRs focused — one topic each. Reference the issue.
3. Normative changes are versioned: the spec is **v1** today and changes stay
   backward compatible where possible. A breaking change bumps the version and
   is gated behind a new format identifier (e.g. a new dropfile format), never a
   silent redefinition. The maintainer decides version bumps.

Note: this is a curated standard maintained by CryptoJones. Proposals are
reviewed (and may be declined to keep the spec small and coherent) — opening an
issue first saves you wasted work.

## Standards for examples

Examples are teaching code and must stay correct:

- **They must run.** Go examples must `go build` and be `gofmt`-clean; shell
  examples must pass `sh -n` (and ideally `shellcheck`).
- **Terminal-only.** A door is a text-mode **VT100/ANSI terminal** program — no
  GUI, window, mouse, or HTTP (see `SPEC.md` §3). Examples must reflect that:
  stdin/stdout (subprocess) or a socket (resident), ANSI for color, `\r\n` line
  endings.
- **Minimal & commented.** Show one model clearly; explain the contract inline.
- **Safe.** Treat all input as hostile, stay in your working/share dir, exit
  cleanly on EOF/disconnect.

## Verifying locally

```sh
# Go examples
gofmt -l examples/                      # prints nothing when clean
( cd examples && go vet ./... )         # if you add a go.mod, or build directly:
go build -o /dev/null examples/resident-skeleton.go

# Shell examples
sh -n examples/hello-door.sh
```

To smoke-test a door end to end, register it on a local AdmiralBBS node
(SysOp panel → Content → register door) and dial in, or for a resident door run
the server and point a `resident` door at its address.

## Showcase your door

Built something runners should play? Open an issue titled `showcase: <name>`
with a one-line description, the door model (subprocess/resident), and a link.
Cool doors may be listed in the README.

## Code of conduct

Be decent. This is a fun homage to BBS culture — keep it welcoming, assume good
faith, and skip the flame wars. Harassment isn't tolerated.

---
*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
