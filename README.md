# Sentinel

Steam game emulator manager built with Wails 3.

## Prerequisites

- **Go 1.26+** - Required for Wails 3
- **Node.js 18+** - For frontend development
- **npm** - Package manager for frontend dependencies

## Installation

### Install Wails 3 CLI

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

Verify installation:

```bash
wails3 version
```

### Install Dependencies

```bash
# Install Go dependencies
go mod tidy

# Install frontend dependencies
cd frontend
npm install
cd ..
```

## Development

Start the development server with hot reload:

```bash
task dev
```

This will:
1. Start the Go backend
2. Start the Vite dev server
3. Open the application window
4. Enable hot reload for both backend and frontend changes

## Building

Build a production binary:

```bash
task build
```

The output binary will be placed in the `bin/` directory.

### Platform-Specific Builds

```bash
# macOS
task build:darwin

# Windows
task build:windows

# Linux
task build:linux
```

## Project Structure

```
sentinel/
├── backend/           # Go backend code
│   ├── app.go        # Main application service
│   ├── config/       # Configuration management
│   ├── cache/        # Caching layer
│   ├── scanner/      # Game scanner
│   ├── steam/        # Steam API integration
│   └── i18n/         # Internationalization
├── frontend/         # React frontend
│   ├── src/
│   │   ├── pages/    # Page components
│   │   └── assets/   # Static assets
│   ├── bindings/     # Generated TypeScript bindings
│   └── package.json
├── build/            # Build configuration
│   ├── config.yml    # Wails 3 configuration
│   ├── darwin/       # macOS-specific files
│   ├── windows/      # Windows-specific files
│   └── linux/        # Linux-specific files
├── main.go           # Application entry point
├── go.mod            # Go module definition
└── Taskfile.yml      # Build tasks
```

## TypeScript Bindings

TypeScript bindings are automatically generated from Go code. To regenerate:

```bash
task generate:bindings
```

The bindings are generated in `frontend/bindings/` and can be imported as:

```typescript
import { LoadConfig, AddEmulator } from '@wa/sentinel/backend/config/cfgfile';
import { CfgFile, Emulator } from '@wa/sentinel/backend/config/models';
```

## Configuration

Application configuration is stored in `build/config.yml`. This file contains:

- Application metadata (name, description, version)
- Development mode settings
- File associations
- Platform-specific settings

## Cleaning Build Artifacts

```bash
task clean
```

This removes:
- Frontend dist directory
- Generated binaries
- Build cache

## License

[Your License Here]
