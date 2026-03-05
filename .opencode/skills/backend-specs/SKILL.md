---
name: backend-specs
description: Rules for creating and editing backend components with Go and Wails. Use when making changes to the Go section
---

The backend is written in Go using the Wails framework. It is located in the root directory and `internal/`.

## Tech Stack
- **Language**: Go
- **Framework**: Wails v3

## Project Structure
- `main.go`: The entry point of the application. Configures the Wails application options (window size, assets, etc.) and binds the `App` struct.
- `internal/`: Contains the private application code.
  - `internal/app.go`: Defines the `App` struct and its methods. Methods exported on this struct are bound to the frontend.

## Development Guidelines
- **Binding Methods**: To expose a method to the frontend, define it as an exported method (Capitalized) on the `App` struct in `internal/app.go`.
- **Context**: The `startup` method in `internal/app.go` captures the `context.Context` which is needed for runtime interactions (events, dialogs, etc.).
- **Separation of Concerns**: Keep business logic designated to specific domains within `internal/` subpackages if the application grows.
- **Formatting**: Always run `gofmt` (or let your IDE do it) before committing.

## Useful Commands
- `wails dev`: Run the application in development mode.
- `wails build`: Build the application for production.
