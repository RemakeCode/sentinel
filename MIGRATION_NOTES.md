# Wails 2 to Wails 3 Migration Notes

This document describes the migration process from Wails 2 to Wails 3 for the Sentinel project.

## Migration Date

February 2026

## Key Changes

### 1. Architecture

**Wails 2:**
- Used `wails.Run()` API in main.go
- Bindings defined with `//wails:bind` directives
- Configuration via `wails.json`

**Wails 3:**
- Uses `application.New()` with Services binding
- Bindings done via `application.NewService()` in main.go
- Configuration via `Taskfile.yml` and `build/config.yml`
- CLI-based architecture

### 2. Dependencies

**go.mod:**
- Removed: `github.com/wailsapp/wails/v2 v2.10.1`
- Added: `github.com/wailsapp/wails/v3 v3.0.0-alpha.72`
- Go version updated: 1.24.0 → 1.26.0

**frontend/package.json:**
- Removed: `@wailsapp/runtime`
- Added: `@wailsio/runtime: ^3.0.0-alpha.72`

### 3. Main.go Changes

**Before (Wails 2):**
```go
package main

import (
    "github.com/wailsapp/wails/v2"
    "github.com/wailsapp/wails/v2/pkg/options"
    "github.com/wailsapp/wails/v2/pkg/options/assetserver"
    "sentinel/backend"
)

func main() {
    err := wails.Run(&options.App{
        Title:  "Sentinel",
        Width:  1024,
        Height: 768,
        AssetServer: &assetserver.Options{
            Assets: embed.FS,
        },
        OnStartup:  backend.Startup,
        OnShutdown: backend.Shutdown,
        Bind: []interface{}{
            backend.App{},
            config.CfgFile{},
        },
    })
    if err != nil {
        println("Error:", err.Error())
    }
}
```

**After (Wails 3):**
```go
package main

import (
    "embed"
    "io/fs"
    "log"

    "github.com/wailsapp/wails/v3/pkg/application"
    "github.com/wailsapp/wails/v3/pkg/assetserver"
    "sentinel/backend"
    "sentinel/backend/config"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
    distFS, err := fs.Sub(assets, "frontend/dist")
    if err != nil {
        log.Fatal(err)
    }

    app := application.New(application.Options{
        Name:        "Sentinel",
        Description: "Steam game emulator manager",
        Services: []application.Service{
            application.NewService(&backend.App{}),
            application.NewService(&config.CfgFile{}),
        },
        Assets: application.AssetOptions{
            Handler: application.AssetFileServerFS(distFS),
        },
        Mac: application.MacOptions{
            ApplicationShouldTerminateAfterLastWindowClosed: true,
        },
    })

    app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "Sentinel",
        Width:  1024,
        Height: 768,
        URL:    "/",
    })

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 4. Backend Changes

**backend/config/config.go:**
- Removed: `import "github.com/wailsapp/wails/v2/pkg/runtime"`
- Removed: `runtime.LogErrorf()` calls
- Modified `SelectDirectory()` to return empty string (file dialogs now handled via frontend runtime)

### 5. Frontend Changes

**TypeScript Bindings:**
- Old path: `@wa/go/config/CfgFile`
- New path: `@wa/sentinel/backend/config/cfgfile`
- Generated with: `wails3 generate bindings`

**Import Example:**
```typescript
import { LoadConfig, AddEmulator, RemoveEmulator, ToggleEmulatorNotification, SelectDirectory } from '@wa/sentinel/backend/config/cfgfile';
import { CfgFile, Emulator } from '@wa/sentinel/backend/config/models';
```

**Property Names:**
- `appConfig?.Emulators` → `appConfig?.emulators` (lowercase)

**vite.config.ts:**
Added path alias for bindings:
```typescript
resolve: {
  alias: {
    '@wa': path.resolve(__dirname, './bindings')
  }
}
```

### 6. Build System

**New Files:**
- `Taskfile.yml` - Main build tasks
- `build/config.yml` - Wails 3 configuration
- `build/Taskfile.yml` - Common build tasks
- `build/darwin/Taskfile.yml` - macOS-specific tasks
- `build/windows/Taskfile.yml` - Windows-specific tasks
- `build/linux/Taskfile.yml` - Linux-specific tasks

**Removed Files:**
- `wails.json` - No longer needed in Wails 3

**Build Commands:**
- `task dev` - Development server
- `task build` - Production build
- `task clean` - Clean artifacts
- `task generate:bindings` - Generate TypeScript bindings

### 7. Asset Embedding

Added to main.go:
```go
//go:embed all:frontend/dist
var assets embed.FS
```

### 8. TypeScript Configuration

**frontend/tsconfig.json:**
```json
{
  "compilerOptions": {
    "paths": {
      "@wa/*": ["./bindings/*"]
    }
  }
}
```

## Known Issues

1. **Icon Generation**: The icon generation task in `build/darwin/Taskfile.yml` is commented out due to missing `appicon.icon` directory. This needs to be addressed before production builds.

2. **File Dialogs**: The `SelectDirectory()` method in `backend/config/config.go` currently returns an empty string. File dialogs should be handled via the frontend runtime using `@wailsio/runtime`.

## Testing Checklist

- [ ] Test adding new emulator paths
- [ ] Test listing all emulators
- [ ] Test removing user-added emulators
- [ ] Test toggling emulator notifications
- [ ] Test browsing for emulator directory
- [ ] Test on target platforms (macOS, Windows, Linux)
- [ ] Performance testing and comparison

## Resources

- [Wails 3 Documentation](https://v3alpha.wails.io/)
- [Wails 3 Quick Start](https://v3alpha.wails.io/quick-start/)
- [Wails 3 Migration Guide](https://v3alpha.wails.io/docs/migration/)
- [Zero Project Template](https://github.com/wailsapp/wails-template-zero) - Reference for Wails 3 structure

## Rollback Strategy

If issues arise, rollback can be done by:
1. Checking out the pre-migration commit
2. Restoring `wails.json` from git history
3. Reverting `go.mod` and `package.json` changes
4. Running `go mod tidy` and `npm install`
