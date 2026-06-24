#!/bin/sh
# hello-door.sh — a minimal ABBS *subprocess* door.
#
# Demonstrates the whole subprocess contract (see SPEC.md §1):
#   - read caller context from the door32.sys dropfile ($DOORFILE)
#   - honor ANSI vs ASCII (dropfile line 10)
#   - talk over stdin/stdout with \r\n line endings
#   - exit cleanly when the caller quits or stdin hits EOF
#
# Register from the SysOp panel: Content -> register door -> subprocess,
# command = /absolute/path/to/hello-door.sh

dropfile="${DOORFILE:-door32.sys}"
field() { sed -n "${1}p" "$dropfile" 2>/dev/null | tr -d '\r'; }

handle=$(field 7)
[ -z "$handle" ] && handle="runner"
emu=$(field 10)        # 0 = ASCII, 1 = ANSI
node=$(field 11)

if [ "$emu" = "1" ]; then C='\033[1;36m'; R='\033[0m'; else C=''; R=''; fi

printf '%bWelcome to Hello Door, %s! (node %s)%b\r\n' "$C" "$handle" "$node" "$R"
printf 'Type something and press enter. Q to quit.\r\n'

# Raw input: read a line at a time; `read` gives us CR/LF handling on most
# shells. A real door should also handle backspace if it echoes characters.
while printf '> ' && IFS= read -r line; do
  line=$(printf '%s' "$line" | tr -d '\r')
  case "$line" in
    [Qq]|quit|QUIT) printf 'Jacking out. NO CARRIER\r\n'; exit 0 ;;
    '') : ;;
    *) printf 'You said: %s\r\n' "$line" ;;
  esac
done
# stdin closed (caller disconnected) -> fall through and exit cleanly.
exit 0
