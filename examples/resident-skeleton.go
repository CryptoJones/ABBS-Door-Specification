// resident-skeleton.go — a minimal ABBS *resident* (multiplayer) door server.
//
// Demonstrates the resident contract (see SPEC.md §2): listen on TCP, accept
// many callers, and drive each one's terminal over a transparent byte stream.
// The BBS bridges each caller to THIS one process, so all connected callers
// share whatever world you build here.
//
// This skeleton implements a trivial shared "lobby" with a name prompt, a
// broadcast `say`, a `who` list, and `quit` — enough to show the shape. Grow it
// into a real game; the production reference is Console Cowboy 2026 in the
// AdmiralBBS repo (src/game/cowboy + src/cmd/cowboy).
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
	"sort"
	"strings"
	"sync"
)

type player struct {
	name string
	out  chan string
}

type lobby struct {
	mu      sync.Mutex
	players map[*player]struct{}
}

func (l *lobby) join(p *player)  { l.mu.Lock(); l.players[p] = struct{}{}; l.mu.Unlock() }
func (l *lobby) leave(p *player) { l.mu.Lock(); delete(l.players, p); l.mu.Unlock() }

func (l *lobby) broadcast(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for p := range l.players {
		select {
		case p.out <- msg: // non-blocking: never let one slow caller stall the rest
		default:
		}
	}
}

func (l *lobby) names() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for p := range l.players {
		out = append(out, p.name)
	}
	sort.Strings(out)
	return out
}

func main() {
	addr := flag.String("addr", "127.0.0.1:4001", "TCP listen address for the BBS bridge")
	flag.Parse()

	l := &lobby{players: map[*player]struct{}{}}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("resident door listening on %s", *addr)
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go serve(c, l)
	}
}

func serve(c net.Conn, l *lobby) {
	defer c.Close()
	r := bufio.NewReader(c)
	p := &player{out: make(chan string, 64)}

	// Writer goroutine: drains the player's output queue to the socket.
	done := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-p.out:
				if _, err := c.Write([]byte(s)); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	c.Write([]byte("\r\nResident lobby. Enter your handle: "))
	name, err := readLine(r)
	if err != nil || strings.TrimSpace(name) == "" {
		close(done)
		return
	}
	p.name = strings.TrimSpace(name)
	l.join(p)
	p.out <- fmt.Sprintf("Welcome, %s. Commands: say <msg>, who, quit.\r\n", p.name)
	l.broadcast(fmt.Sprintf("* %s jacks in.\r\n", p.name))

	for {
		p.out <- "> "
		line, err := readLine(r)
		if err != nil {
			break // caller disconnected — exit cleanly
		}
		line = strings.TrimSpace(line)
		switch {
		case line == "quit":
			p.out <- "NO CARRIER\r\n"
			goto out
		case line == "who":
			p.out <- "Online: " + strings.Join(l.names(), ", ") + "\r\n"
		case strings.HasPrefix(line, "say "):
			l.broadcast(fmt.Sprintf("%s: %s\r\n", p.name, strings.TrimPrefix(line, "say ")))
		case line == "":
		default:
			p.out <- "Unknown. Try: say <msg>, who, quit.\r\n"
		}
	}
out:
	l.leave(p)
	l.broadcast(fmt.Sprintf("* %s jacks out.\r\n", p.name))
	close(done)
}

// readLine reads one line of raw terminal input, handling CR/LF and backspace.
func readLine(r *bufio.Reader) (string, error) {
	var b []byte
	for {
		ch, err := r.ReadByte()
		if err != nil {
			return string(b), err
		}
		switch ch {
		case '\r', '\n':
			if ch == '\r' { // swallow a paired LF
				if nb, e := r.ReadByte(); e == nil && nb != '\n' {
					_ = r.UnreadByte()
				}
			}
			return string(b), nil
		case 0x08, 0x7f:
			if len(b) > 0 {
				b = b[:len(b)-1]
			}
		default:
			if ch >= 0x20 && ch < 0x7f {
				b = append(b, ch)
			}
		}
	}
}
