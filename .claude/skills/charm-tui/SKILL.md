---
name: charm-tui
description: Use when writing, reviewing, or modifying TUI code using the Charm ecosystem (Bubble Tea, Bubbles, Lip Gloss). Triggers on tasks involving terminal UI components, TUI styling, interactive terminal apps, or Bubble Tea models/updates/views.
user-invocable: true
argument-hint: [task-description]
---

# Charm TUI Ecosystem Guide

Use this skill when building terminal user interfaces with the Charm stack. Apply these patterns, components, and APIs to write idiomatic Charm code.

## Bubble Tea — The Elm Architecture for Terminals

**Repo**: https://github.com/charmbracelet/bubbletea
**Docs**: https://pkg.go.dev/github.com/charmbracelet/bubbletea
**Tutorials**: [Basics](https://github.com/charmbracelet/bubbletea/tree/main/tutorials/basics) | [Commands](https://github.com/charmbracelet/bubbletea/tree/main/tutorials/commands)
**All examples**: https://github.com/charmbracelet/bubbletea/tree/main/examples

Every Bubble Tea app implements three methods on a model:

```go
type model struct { /* your state */ }

// Init returns an initial command (or nil for none)
func (m model) Init() tea.Cmd { return nil }

// Update handles messages, returns updated model + optional command
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    }
    return m, nil
}

// View renders UI as a string — called after every Update
func (m model) View() string { return "Hello, World!\n" }

func main() {
    p := tea.NewProgram(model{})
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Key types

| Type | Purpose |
|------|---------|
| `tea.Model` | Interface: `Init()`, `Update()`, `View()` |
| `tea.Cmd` | Function that performs I/O and returns a `tea.Msg` |
| `tea.Msg` | Any type — events flow through `Update()` |
| `tea.KeyMsg` | Keyboard input; use `.String()` for key name |
| `tea.WindowSizeMsg` | Terminal resize event with `.Width`, `.Height` |
| `tea.MouseMsg` | Mouse events (when enabled) |
| `tea.Batch(cmds...)` | Combine multiple `Cmd`s into one |
| `tea.Quit` | Command to exit the program |
| `tea.ClearScreen` | Command to clear terminal |
| `tea.EnterAltScreen` | Command to enter alt screen buffer |
| `tea.ExitAltScreen` | Command to leave alt screen buffer |

### Patterns

- **State machines**: Use an enum/int field on the model to track which "screen" or phase you're in. Switch on it in both `Update()` and `View()`.
- **Sub-models**: Compose by embedding Bubbles components in your model. Forward messages to them in `Update()` and call their `View()` in your `View()`.
- **Commands**: Never do I/O directly in `Update()`. Return a `tea.Cmd` instead. The runtime calls it asynchronously and delivers the result as a `tea.Msg`.
- **Batch commands**: Use `tea.Batch(cmd1, cmd2)` when you need to fire multiple commands from a single `Update()`.
- **Quitting**: Return `tea.Quit` from `Update()` to exit.

### Debugging

```go
// Log to file (call early in main, defer f.Close())
f, err := tea.LogToFile("debug.log", "debug")

// Use Delve headless for stepping
// dlv debug --headless --api-version=2 --listen=127.0.0.1:43000
```

---

## Bubbles — Pre-built Components

**Repo**: https://github.com/charmbracelet/bubbles
**Docs**: https://pkg.go.dev/github.com/charmbracelet/bubbles

Each Bubble is a `tea.Model` itself. Embed it, forward messages, render its view.

### Available Components

#### Text Input (single-line)
`bubbles/textinput` — Single-line text field. Supports unicode, pasting, cursor movement, placeholder text, character limits, input masking (passwords).
- [Source](https://github.com/charmbracelet/bubbles/tree/master/textinput)
- [Example: single field](https://github.com/charmbracelet/bubbletea/blob/main/examples/textinput/main.go)
- [Example: multiple fields](https://github.com/charmbracelet/bubbletea/blob/main/examples/textinputs/main.go)

```go
ti := textinput.New()
ti.Placeholder = "Type here..."
ti.Focus()
ti.CharLimit = 156
ti.Width = 20
```

#### Text Area (multi-line)
`bubbles/textarea` — Multi-line text input. Supports unicode, pasting, vertical scrolling, line numbers, character limits.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/textarea)
- [Example: chat](https://github.com/charmbracelet/bubbletea/blob/main/examples/chat/main.go)
- [Example: story](https://github.com/charmbracelet/bubbletea/blob/main/examples/textarea/main.go)

```go
ta := textarea.New()
ta.Placeholder = "Tell your story..."
ta.Focus()
```

#### Spinner
`bubbles/spinner` — Animated loading indicator with many built-in styles.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/spinner)
- [Example: basic](https://github.com/charmbracelet/bubbletea/blob/main/examples/spinner/main.go)
- [Example: multiple styles](https://github.com/charmbracelet/bubbletea/blob/main/examples/spinners/main.go)

Built-in spinner types: `spinner.Line`, `spinner.Dot`, `spinner.MiniDot`, `spinner.Jump`, `spinner.Pulse`, `spinner.Points`, `spinner.Globe`, `spinner.Moon`, `spinner.Monkey`, `spinner.Meter`, `spinner.Hamburger`, `spinner.Ellipsis`.

```go
s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
```

#### List
`bubbles/list` — Full-featured list with pagination, fuzzy filtering, help text, spinner, status bar. Batteries-included.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/list)
- [Example: default](https://github.com/charmbracelet/bubbletea/blob/main/examples/list-default/main.go)
- [Example: simple](https://github.com/charmbracelet/bubbletea/blob/main/examples/list-simple/main.go)
- [Example: fancy](https://github.com/charmbracelet/bubbletea/blob/main/examples/list-fancy/main.go)

Items must implement `list.Item` interface (`FilterValue() string`). Use `list.DefaultDelegate` or implement `list.ItemDelegate` for custom rendering.

```go
items := []list.Item{item{title: "Foo", desc: "Bar"}}
l := list.New(items, list.NewDefaultDelegate(), 20, 14)
l.Title = "My List"
```

#### Table
`bubbles/table` — Tabular data display with column headers, row navigation, vertical scrolling.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/table)
- [Example](https://github.com/charmbracelet/bubbletea/blob/main/examples/table/main.go)

```go
columns := []table.Column{
    {Title: "Name", Width: 20},
    {Title: "Age", Width: 5},
}
rows := []table.Row{{"Alice", "30"}, {"Bob", "25"}}
t := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithHeight(7))
```

#### Viewport
`bubbles/viewport` — Scrollable content pane. Mouse support, high-performance mode, pager-style keybindings.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/viewport)
- [Example: pager](https://github.com/charmbracelet/bubbletea/blob/main/examples/pager/main.go)

```go
vp := viewport.New(80, 20)
vp.SetContent(longString)
```

#### Progress Bar
`bubbles/progress` — Animated or static progress meter. Solid fill, gradient fill, configurable.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/progress)
- [Example: animated](https://github.com/charmbracelet/bubbletea/blob/main/examples/progress-animated/main.go)
- [Example: static](https://github.com/charmbracelet/bubbletea/blob/main/examples/progress-static/main.go)

```go
p := progress.New(progress.WithDefaultGradient())
// In View():  p.ViewAs(0.5)  // 50%
```

#### Paginator
`bubbles/paginator` — Page navigation. Dot-style (iOS-like) or Arabic numerals.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/paginator)
- [Example](https://github.com/charmbracelet/bubbletea/blob/main/examples/paginator/main.go)

```go
p := paginator.New()
p.Type = paginator.Dots // or paginator.Arabic
p.PerPage = 10
p.SetTotalPages(len(items))
```

#### Help
`bubbles/help` — Auto-generated keybinding help bar. Single-line and full modes.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/help)
- [Example](https://github.com/charmbracelet/bubbletea/blob/main/examples/help/main.go)

Keybindings implement `key.Binding` from `bubbles/key`. Group them with `help.KeyMap` interface.

```go
h := help.New()
// In View(): h.View(myKeyMap)
```

#### Key Bindings
`bubbles/key` — Non-visual keybinding definitions. Used with the help bubble for auto-generated docs. Supports enable/disable.

```go
binding := key.NewBinding(
    key.WithKeys("q", "esc"),
    key.WithHelp("q/esc", "quit"),
)
```

#### File Picker
`bubbles/filepicker` — Directory browser with file selection and extension filtering.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/filepicker)
- [Example](https://github.com/charmbracelet/bubbletea/blob/main/examples/file-picker/main.go)

```go
fp := filepicker.New()
fp.AllowedTypes = []string{".go", ".mod"}
fp.CurrentDirectory, _ = os.Getwd()
```

#### Timer
`bubbles/timer` — Countdown timer with configurable tick interval.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/timer)
- [Example](https://github.com/charmbracelet/bubbletea/blob/main/examples/timer/main.go)

```go
t := timer.NewWithInterval(5*time.Second, time.Millisecond)
```

#### Stopwatch
`bubbles/stopwatch` — Count-up timer with configurable tick interval.
- [Source](https://github.com/charmbracelet/bubbles/tree/master/stopwatch)
- [Example](https://github.com/charmbracelet/bubbletea/blob/main/examples/stopwatch/main.go)

```go
sw := stopwatch.NewWithInterval(time.Millisecond)
```

---

## Lip Gloss — Styling & Layout

**Repo**: https://github.com/charmbracelet/lipgloss
**Docs**: https://pkg.go.dev/github.com/charmbracelet/lipgloss

### Creating Styles

```go
style := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    Padding(0, 1).
    MarginTop(1)

output := style.Render("Hello!")
```

Styles are immutable value types — chaining returns a new copy.

### Colors

```go
lipgloss.Color("5")          // ANSI 16
lipgloss.Color("86")         // ANSI 256
lipgloss.Color("#FF6AD5")    // True color

// Adapt to light/dark terminal backgrounds
lipgloss.AdaptiveColor{Light: "236", Dark: "248"}

// Specify exact values per color profile
lipgloss.CompleteColor{TrueColor: "#FF6AD5", ANSI256: "86", ANSI: "5"}
```

### Text formatting

`.Bold(true)`, `.Italic(true)`, `.Faint(true)`, `.Underline(true)`, `.Strikethrough(true)`, `.Blink(true)`, `.Reverse(true)`

### Block-level formatting

```go
// Padding and margins (CSS shorthand order: top, right, bottom, left)
.Padding(1, 2)       // vertical 1, horizontal 2
.Margin(1, 2, 1, 2)  // all four sides

// Dimensions
.Width(40)            // minimum width
.Height(5)            // minimum height
.MaxWidth(80)         // enforce max
.MaxHeight(20)

// Alignment within the block
.Align(lipgloss.Center)
.Align(lipgloss.Left)
.Align(lipgloss.Right)
```

### Borders

```go
.Border(lipgloss.RoundedBorder())   // ╭─╮│ │╰─╯
.Border(lipgloss.NormalBorder())     // ┌─┐│ │└─┘
.Border(lipgloss.ThickBorder())      // ┏━┓┃ ┃┗━┛
.Border(lipgloss.DoubleBorder())     // ╔═╗║ ║╚═╝

// Selective borders
.Border(lipgloss.NormalBorder(), true, false, true, false) // top + bottom only
.BorderForeground(lipgloss.Color("63"))
```

### Layout — Joining

```go
// Horizontal: align tops, centers, or bottoms
lipgloss.JoinHorizontal(lipgloss.Top, left, right)
lipgloss.JoinHorizontal(lipgloss.Center, a, b, c)
lipgloss.JoinHorizontal(lipgloss.Bottom, a, b)

// Vertical: align left, center, or right
lipgloss.JoinVertical(lipgloss.Left, top, bottom)
lipgloss.JoinVertical(lipgloss.Center, a, b)
```

### Layout — Placement

```go
// Place content within a sized region
lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
lipgloss.PlaceVertical(height, lipgloss.Center, content)
```

### Measurement

```go
w := lipgloss.Width(rendered)    // width of rendered block
h := lipgloss.Height(rendered)   // height of rendered block
w, h := lipgloss.Size(rendered)  // both
```

### Sub-packages

#### Tables (`lipgloss/table`)
- [Source & examples](https://github.com/charmbracelet/lipgloss/tree/master/examples/table)

```go
t := table.New().
    Border(lipgloss.NormalBorder()).
    BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
    Headers("NAME", "AGE").
    Row("Alice", "30").
    Row("Bob", "25")

fmt.Println(t)
```

Per-cell styling with `StyleFunc`:
```go
t.StyleFunc(func(row, col int) lipgloss.Style {
    if row == 0 { return headerStyle }
    if row%2 == 0 { return evenRowStyle }
    return oddRowStyle
})
```

#### Lists (`lipgloss/list`)

```go
l := list.New("Item A", "Item B", "Item C").
    Enumerator(list.Bullet)  // or list.Arabic, list.Alphabet, list.Roman, list.Tree

// Nested
l := list.New("Parent", list.New("Child A", "Child B"))
```

#### Trees (`lipgloss/tree`)

```go
t := tree.Root("Root").
    Child("Branch A").
    Child(
        tree.Root("Branch B").
            Child("Leaf 1").
            Child("Leaf 2"),
    )
```

Enumerators: `tree.DefaultEnumerator`, `tree.RoundedEnumerator`

---

## Companion Libraries

| Library | Purpose | Link |
|---------|---------|------|
| Harmonica | Spring-based animation | https://github.com/charmbracelet/harmonica |
| BubbleZone | Mouse click zones | https://github.com/lrstanley/bubblezone |
| Glamour | Markdown rendering | https://github.com/charmbracelet/glamour |
| Termenv | Low-level terminal colors | https://github.com/muesli/termenv |
| Reflow | ANSI-aware word wrap/truncate | https://github.com/muesli/reflow |
| ntcharts | Terminal charts | https://github.com/NimbleMarkets/ntcharts |

---

## Common Patterns

### Composing sub-models

```go
type model struct {
    input    textinput.Model
    spinner  spinner.Model
    // ... your state
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd

    m.input, cmd = m.input.Update(msg)
    cmds = append(cmds, cmd)

    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}
```

### State machine with views

```go
type state int

const (
    stateInput state = iota
    stateLoading
    stateResult
)

type model struct {
    state state
    // sub-models...
}

func (m model) View() string {
    switch m.state {
    case stateInput:
        return m.input.View()
    case stateLoading:
        return m.spinner.View() + " Loading..."
    case stateResult:
        return m.viewport.View()
    }
    return ""
}
```

### Responsive layout

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.viewport.Width = msg.Width
        m.viewport.Height = msg.Height - headerHeight - footerHeight
    }
    // ...
}
```

When implementing TUI features for $ARGUMENTS, apply these patterns and choose the appropriate components from the catalog above.
