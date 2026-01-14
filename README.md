# goasciinema

A fast terminal session recorder written in Go. This is a Go implementation of [asciinema](https://asciinema.org), optimized for faster startup and execution compared to the Python original.

## Installation

### From source

```bash
go install github.com/ober/goasciinema@latest
```

### Build locally

```bash
git clone https://github.com/ober/goasciinema.git
cd goasciinema
go build -o goasciinema .
```

## Usage

### Record a terminal session

```bash
goasciinema rec demo.cast
```

Options:
- `-c, --command` - Command to record (default: `$SHELL`)
- `-t, --title` - Title of the recording
- `-i, --idle-time-limit` - Limit recorded idle time to given seconds
- `--stdin` - Enable stdin recording
- `--append` - Append to existing recording
- `--cols` - Override terminal columns
- `--rows` - Override terminal rows
- `-q, --quiet` - Quiet mode (suppress notices)
- `-y, --overwrite` - Overwrite existing file without asking

### Play a recording

```bash
goasciinema play demo.cast
```

Options:
- `-s, --speed` - Playback speed (e.g., 2 for 2x speed)
- `-i, --idle-time-limit` - Limit replayed idle time to given seconds
- `-m, --maxwait` - Maximum wait time between frames
- `-l, --loop` - Loop playback

### Print full output

```bash
goasciinema cat demo.cast
```

Outputs all terminal output without any timing, useful for extracting raw content.

### Upload to asciinema.org

```bash
goasciinema upload demo.cast
```

### Link to your account

```bash
goasciinema auth
```

## Configuration

Configuration is loaded from:
1. `$ASCIINEMA_CONFIG_HOME/config`
2. `$XDG_CONFIG_HOME/asciinema/config`
3. `~/.config/asciinema/config`

Example config file (INI format):

```ini
[api]
url = https://asciinema.org

[record]
command = /bin/bash
stdin = no
idle_time_limit = 2.0
quiet = no

[play]
speed = 1.0
idle_time_limit = 2.0
maxwait = 2.0
```

Environment variables:
- `ASCIINEMA_API_URL` - Override API URL
- `ASCIINEMA_CONFIG_HOME` - Override config directory
- `ASCIINEMA_INSTALL_ID` - Override install ID

## File Format

goasciinema uses the [asciicast v2](https://docs.asciinema.org/manual/asciicast/v2/) format:

```
{"version": 2, "width": 80, "height": 24, "timestamp": 1234567890, ...}
[0.0, "o", "Hello "]
[0.5, "o", "World!\r\n"]
```

## License

MIT

## Credits

This is a Go reimplementation of [asciinema](https://github.com/asciinema/asciinema) by Marcin Kulik.
