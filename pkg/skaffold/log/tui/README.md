# TUI Logger for Skaffold

This package provides a Terminal User Interface (TUI) for viewing logs from multiple Kubernetes containers in a more organized way.

## Features

- **Multiple Log Sources**: View logs from different pods/containers in separate views
- **Color-Coded**: Each container gets its own color for easy identification
- **Keyboard Navigation**: 
  - `↑/↓`: Navigate between log sources in the list
  - `Enter`: Select a log source to view
  - `Tab`: Switch between source list and log view
  - `Ctrl+C`: Exit the TUI
- **Auto-Scroll**: Logs automatically scroll to show the latest entries
- **Buffered Logs**: Keeps the last 10,000 lines per source to prevent memory issues

## Usage

Use the `--tui` flag with any Skaffold command that supports log tailing:

```bash
# Development mode with TUI
skaffold dev --tui

# Run mode with TUI
skaffold run --tui

# Debug mode with TUI
skaffold debug --tui

# Deploy with TUI
skaffold deploy --tui
```

The `--tui` flag automatically enables `--tail`, so logs will be streamed from deployed pods.

## Architecture

- **TUILogger**: Main component that manages the terminal UI using the `tview` library
- **LogSource**: Represents a single source of logs (pod/container combination)
- **TUIWriter**: Implements `io.Writer` to redirect logs to the TUI

## Limitations

- Requires a terminal that supports ANSI escape codes
- Not suitable for CI/CD environments (use standard `--tail` instead)
- Tests require a TTY and are skipped in non-interactive environments

## Implementation Details

The TUI is built using:
- `github.com/rivo/tview` - Terminal UI framework
- `github.com/gdamore/tcell/v2` - Terminal handling library

Both libraries are already included in Skaffold's vendor directory.
