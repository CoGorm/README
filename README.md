# readme

Read a README in your terminal, the way it was meant to look.

`readme` renders markdown with [glamour](https://github.com/charmbracelet/glamour)
and pages it with [bubbletea](https://github.com/charmbracelet/bubbletea) — syntax
highlighting, tables, emoji, and a searchable scrollback, without leaving the shell.

## Install

Needs Go 1.25.8 or newer, which `GOTOOLCHAIN=auto` (the default) will fetch for you.

```sh
go install github.com/CoGorm/README/cmd/readme@latest
```

Or build from a checkout:

```sh
go build -o readme ./cmd/readme
```

## Usage

```sh
readme                # find the readme in the current directory
readme docs/api.md    # render a specific file
readme ../other-repo  # find the readme in another directory
readme CONTRIBUTING   # the extension is optional
cat notes.md | readme # read from a pipe
```

With no argument it looks for `README.md` in the current directory, then
`readme.md`, `README`, and the other usual spellings, then falls back to
`.github/`, `docs/`, and `doc/`.

### Flags

| Flag | Meaning |
| --- | --- |
| `-s`, `--style` | `auto`, `dark`, `light`, `dracula`, `tokyo-night`, `pink`, `ascii`, `notty`, or a path to a glamour style JSON file |
| `-w`, `--width` | wrap width in columns (default: terminal width, capped at 100) |
| `-n`, `--no-pager` | print and exit instead of opening the pager |
| `-v`, `--version` | print the version |
| `-h`, `--help` | show help |

### In the pager

| Keys | Action |
| --- | --- |
| `j` `k` `↑` `↓` | scroll a line |
| `d` `u` | scroll half a page |
| `f` `b` `space` | scroll a page |
| `g` `G` | jump to the top or bottom |
| `/` | search |
| `n` `N` | next or previous match |
| `?` | toggle the key list |
| `q` `esc` | quit |

## Behaviour worth knowing

The pager only opens when it earns its keep: if stdout is not a terminal, or the
document already fits on screen, `readme` prints the rendered text and exits. That
keeps pipes honest:

```sh
readme | grep -i install
readme --style notty > plain.txt
```

Resizing the window reflows the document rather than just re-wrapping the old
layout, and `--style auto` follows your terminal's light or dark background.

Working that background out means asking the terminal over an OSC 11 escape
sequence and waiting for a reply, and plenty of terminals — tmux among them —
never reply. So the query carries a cursor position report behind it: almost
every terminal answers *that*, and its reply means no background colour is
coming, which ends the wait immediately. Failing both, the probe gives up after
250ms and falls back to `$COLORFGBG`. Naming a style skips the question
altogether, so `readme -s dark` never asks at all.

## Development

```sh
go test ./...
```

- `cmd/readme` — flags, input resolution, and the decision to page or print
- `internal/find` — locating a readme on disk
- `internal/render` — markdown to styled text
- `internal/theme` — the bounded terminal background probe
- `internal/tui` — the pager
