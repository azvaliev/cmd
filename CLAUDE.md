# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

`cmd` is a CLI tool that translates natural language queries into terminal commands. It runs a local LLM via llama.cpp, uses RAG to look up known commands before generating, and presents results in an interactive TUI. Currently macOS M-chip only, 16GB RAM minimum.

The design philosophy is "power tool for flow state" — the interaction should feel like a brief detour, not a context switch. Speed over ceremony, minimal UI chrome, user always in control. See `README.md` for full design philosophy, RAG architecture decisions, embedding strategy, and roadmap.

## Development Commands

```bash
# Run in development mode
task dev <args>

# Format code
task fmt

# Build/install
go install github.com/azvaliev/cmd

# Enable debug output (prints llama-server stderr)
DEBUG=1 task dev
```

There are no tests in the codebase yet.

## Architecture

**Bubble Tea (Elm Architecture) TUI** with a 6-state flow:
Generating → Confirm → Explain → Output → Correction → Teaching

**Package layout:**

- `cmd/cmd/` — TUI application (Bubble Tea model, state machine, rendering).

- `internal/pkg/ai/` — LLM backend. Manages the llama.cpp subprocess lifecycle, wraps Firebase Genkit for inference, and detects shell environment. Implements Genkit's `Plugin` interface to expose the local LLM as an OpenAI-compatible provider.

- `internal/pkg/env/` — Environment variable configuration.

**Key dependencies:** Charmbracelet Bubble Tea/Bubbles for TUI, Firebase Genkit for LLM orchestration.

## Conventions

- Llama.cpp binary path and model paths are currently hardcoded in `llama.go` to developer-local paths.
- Resource cleanup uses explicit `Dispose()` methods with SIGTERM for subprocess management.
