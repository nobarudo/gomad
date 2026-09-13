# gomad

A keyboard-centric Markdown viewer for the terminal, built with Go and Bubble Tea.

`gomad` provides a focused, terminal-native environment for reading Markdown documents. It incorporates intuitive Vim-style navigation, an interactive table of contents, incremental search, and multiple built-in color themes.

---

## Features

- **Vim-Style Navigation**  
  Smooth scrolling and jumping between paragraphs or headings using familiar Vim keybindings.

- **Interactive Table of Contents (Sidebar)**  
  A collapsible sidebar displaying document headings hierarchically indented according to their level. Jump directly to sections while keeping the table of contents open.

- **Split-View Focus Switching**  
  Toggle focus between the sidebar and the main content using `h` / `l` or `Tab` to read and navigate concurrently.

- **Incremental Search**  
  Quickly search across documents with visual match counts, highlight navigation, and ANSI-safe pattern matching.

- **Themes & Styling**  
  Beautiful typography and syntax highlighting powered by Glamour. Switch between preset color styles interactively or set a default via command-line flags.

- **Auto-Reload (Live Preview)**  
  Automatically detects changes to the viewed Markdown file (with atomic-save handling and debouncing) and reloads content in real time while preserving your scroll position.

- **In-App Keybinding Help**  
  Quick reference modal available directly within the viewer.

---

## Installation

### Prerequisites

- Go 1.22 or later

### Using `go install` (Recommended)

```bash
go install github.com/nobarudo/gomad@latest
```

> [!NOTE]
> Make sure `$(go env GOPATH)/bin` (typically `~/go/bin`) is included in your system's `PATH`.

### Building from Source

```bash
git clone https://github.com/nobarudo/gomad.git
cd gomad
go build -o gomad .
```

---

## Usage

```bash
# View a local Markdown file
gomad <file.md>

# Read from standard input (pipe)
cat <file.md> | gomad
curl -s https://raw.githubusercontent.com/.../README.md | gomad
```

### Options

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--style` | `-s` | Markdown color theme (`tokyo-night`, `dracula`, `dark`, `light`, `pink`, `notty`) | `tokyo-night` |
| `--watch` | `-w` | Enable auto-reload on file change | `true` |
| `--help` | `-h` | Display help information | |

---

## Keybindings

### Navigation (Content)

| Key | Description |
| :--- | :--- |
| `j` / `↓` | Scroll down one line |
| `k` / `↑` | Scroll up one line |
| `d` / `Ctrl+d` | Scroll down half a page |
| `u` / `Ctrl+u` | Scroll up half a page |
| `f` / `Ctrl+f` | Scroll down full page |
| `b` / `Ctrl+b` | Scroll up full page |
| `g` | Jump to the beginning of document |
| `G` | Jump to the end of document |
| `]` / `[` | Jump to next / previous heading |
| `}` / `{` | Jump to next / previous paragraph |

### Table of Contents (Sidebar)

| Key | Description |
| :--- | :--- |
| `t` | Toggle table of contents sidebar |
| `j` / `k` | Move selection up / down in TOC |
| `Enter` | Jump to the selected heading (keeps sidebar open) |
| `h` / `l` / `Tab` | Switch focus between sidebar and document content |
| `Esc` / `t` / `q` | Close table of contents |

### Search

| Key | Description |
| :--- | :--- |
| `/` | Open search input prompt |
| `Enter` | Execute search query |
| `n` | Jump to the next match |
| `N` | Jump to the previous match |
| `Esc` | Clear search query and highlights |

### General

| Key | Description |
| :--- | :--- |
| `r` | Manually reload file from disk |
| `s` | Open theme picker modal |
| `:` | Show keybindings help modal |
| `q` / `Ctrl+c` | Quit the viewer |

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
