// resident-skeleton.go — a minimal ABBS *resident* (multiplayer) door server.
//
// Demonstrates the resident contract (see SPEC.md §2): listen on TCP, accept
// many callers, and drive each one's terminal over a transparent byte stream.
// The BBS bridges each caller to THIS one process, so all connected callers
// share whatever world you build here.
//
// It implements a trivial shared "lobby" — a handle prompt, a broadcast `say`, a
// `who` list, and `quit` — but does so the RIGHT way, so it is a faithful
// reference for the parts of §2 that are easy to get wrong:
//
//   - §2.2 version handshake: advertise our version as the first bytes on connect.
//   - §2.3 input handling (normative): raw, byte-at-a-time — telnet IAC skip,
//     NON-blocking CR/LF partner swallow, backspace echo (\b \b), printable echo,
//     NUL ignored.
//   - §2.4 managed prompt: async output (another player's `say`, joins/leaves)
//     redraws rather than clobbers what a caller is mid-typing.
//   - §2.5 graceful shutdown: notify callers on SIGINT/SIGTERM before exiting.
//
// Grow it into a real game; the production reference is Chrome Circuit Cowboys,
// its own repo at https://github.com/CryptoJones/ChromeCircuitCowboys.
//
// Run it, then register a resident door pointing at its address:
//
//	go run resident-skeleton.go -addr 127.0.0.1:4001
//	# SysOp panel: Content -> register door -> resident -> tcp 127.0.0.1:4001
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
)

// version is advertised to the BBS host via the §2.2 handshake.
const version = "0.1.0"

// clrLine wipes the current row: CR (return to column 0) + ESC[K (erase to end
// of line). It is the heart of the managed-prompt redraw (§2.4).
const clrLine = "\r\x1b[K"

// conn is one caller. The in-progress input and prompt are mutex-guarded so that
// async output (emit/setPrompt) can redraw them without racing the input echo.
type conn struct {
	nc  net.Conn
	out chan string

	mu     sync.Mutex
	inLine []byte // the caller's un-submitted input
	prompt string // the current prompt, redrawn around async output
	name   string
}

// raw enqueues bytes for the writer goroutine. Non-blocking, so one stalled
// caller can never wedge the server.
func (c *conn) raw(s string) {
	select {
	case c.out <- s:
	default:
	}
}

// emit prints async content (chat, joins, leaves) WITHOUT clobbering what the
// caller is mid-typing (§2.4): wipe the row, print the content, then redraw the
// prompt followed by the in-progress input.
func (c *conn) emit(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.raw(clrLine + s + c.prompt + string(c.inLine))
}

// setPrompt updates the prompt and redraws it with the current input.
func (c *conn) setPrompt(p string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.prompt = p
	c.raw(clrLine + c.prompt + string(c.inLine))
}

type lobby struct {
	mu    sync.Mutex
	conns map[*conn]struct{}
}

func (l *lobby) join(c *conn)  { l.mu.Lock(); l.conns[c] = struct{}{}; l.mu.Unlock() }
func (l *lobby) leave(c *conn) { l.mu.Lock(); delete(l.conns, c); l.mu.Unlock() }

func (l *lobby) broadcast(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for c := range l.conns {
		c.emit(msg) // each caller's copy redraws around their own input
	}
}

func (l *lobby) names() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for c := range l.conns {
		if c.name != "" {
			out = append(out, c.name)
		}
	}
	sort.Strings(out)
	return out
}

func main() {
	addr := flag.String("addr", "127.0.0.1:4001", "TCP listen address for the BBS bridge")
	flag.Parse()

	l := &lobby{conns: map[*conn]struct{}{}}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("resident door %s listening on %s", version, *addr)

	// §2.5: on shutdown, tell connected callers, then exit. A real door would
	// flush/persist here first, with a bounded wait so a stuck save can't hang.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		l.broadcast("\r\n* server going down. NO CARRIER\r\n")
		os.Exit(0)
	}()

	for {
		nc, err := ln.Accept()
		if err != nil {
			continue
		}
		go serve(nc, l)
	}
}

func serve(nc net.Conn, l *lobby) {
	defer nc.Close()
	c := &conn{nc: nc, out: make(chan string, 128)}

	// Writer goroutine: drains the output queue to the socket until it's closed.
	go func() {
		for s := range c.out {
			if _, err := nc.Write([]byte(s)); err != nil {
				return
			}
		}
	}()

	r := bufio.NewReader(nc)

	// §2.2: advertise our version as the VERY FIRST bytes (OSC-framed). The host
	// strips it and shows it on the launch line; a terminal reached directly just
	// swallows the sequence, so a raw nc/telnet session sees nothing.
	c.raw("\x1b]ABBS;version=" + version + "\x07")

	c.raw("\r\nResident lobby. Enter your handle: ")
	name, ok := c.readLine(r)
	if !ok || strings.TrimSpace(name) == "" {
		close(c.out)
		return
	}
	c.name = strings.TrimSpace(name)
	l.join(c)
	c.emit(fmt.Sprintf("Welcome, %s. Commands: say <msg>, who, quit.\r\n", c.name))
	l.broadcast(fmt.Sprintf("* %s jacks in.\r\n", c.name))

	for {
		c.setPrompt("> ")
		line, ok := c.readLine(r)
		if !ok {
			break // caller disconnected — exit cleanly
		}
		switch line = strings.TrimSpace(line); {
		case line == "quit":
			c.emit("NO CARRIER\r\n")
			goto out
		case line == "who":
			c.emit("Online: " + strings.Join(l.names(), ", ") + "\r\n")
		case strings.HasPrefix(line, "say "):
			l.broadcast(fmt.Sprintf("%s: %s\r\n", c.name, strings.TrimPrefix(line, "say ")))
		case line == "":
		default:
			c.emit("Unknown. Try: say <msg>, who, quit.\r\n")
		}
	}
out:
	l.leave(c)
	l.broadcast(fmt.Sprintf("* %s jacks out.\r\n", c.name))
	close(c.out)
}

// readLine reads one submitted line of RAW terminal input, implementing the
// normative §2.3 input contract. It returns ok=false on disconnect. The
// in-progress bytes live in c.inLine (mutex-guarded) so async emit()/setPrompt()
// can redraw them without garbling.
func (c *conn) readLine(r *bufio.Reader) (string, bool) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", false
		}
		switch {
		case b == '\r' || b == '\n':
			// Swallow a CRLF/LFCR partner ONLY if one is already buffered — never
			// block waiting for it, or a lone-CR keystroke would hang.
			if r.Buffered() > 0 {
				if nb, e := r.ReadByte(); e == nil {
					if !((b == '\r' && nb == '\n') || (b == '\n' && nb == '\r')) {
						_ = r.UnreadByte()
					}
				}
			}
			c.mu.Lock()
			line := string(c.inLine)
			c.inLine = c.inLine[:0]
			c.raw("\r\n")
			c.mu.Unlock()
			return line, true
		case b == 0x08 || b == 0x7f: // backspace / DEL
			c.mu.Lock()
			if len(c.inLine) > 0 {
				c.inLine = c.inLine[:len(c.inLine)-1]
				c.raw("\b \b") // erase the glyph on screen
			}
			c.mu.Unlock()
		case b == 0x00:
			// ignore NUL
		case b == 0xff: // telnet IAC — discard it and the following command byte
			_, _ = r.ReadByte()
		default:
			if b >= 0x20 && b < 0x7f { // printable: buffer and echo
				c.mu.Lock()
				c.inLine = append(c.inLine, b)
				c.raw(string(b))
				c.mu.Unlock()
			}
		}
	}
}
