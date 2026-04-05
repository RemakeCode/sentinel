# Automated Release Guide

This project uses [semantic-release](https://semantic-release.gitbook.io/) with [Conventional Commits](https://www.conventionalcommits.org/) to automate versioning and releases.

## How It Works

- Every push to `main` branch runs tests and build checks
- GitHub Release is created manually via "Run Workflow" button
- Commit messages are analyzed to determine version bumps
- Linux packages (AppImage, .deb, .rpm) are built automatically
- GitHub Release is created with changelog
- Version tags are created automatically

## Workflow Triggers

| Event | Jobs Run | Description |
|-------|----------|-------------|
| Push to `main` | test, build-check | Automatic CI on every commit |
| Manual Dispatch | test, release | Creates a new release |

### Manual Release

To create a release:
1. Go to GitHub Actions → CI/CD workflow
2. Click "Run workflow"
3. Select branch and click "Run workflow"

The release job will:
- Run tests first
- Analyze commit messages since last release
- Determine version bump (major/minor/patch)
- Create GitHub Release with changelog
- Upload Linux packages

## Commit Message Format

### Structure

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | Description | Version Bump | Example |
|------|-------------|--------------|---------|
| `feat` | New feature | Minor (1.1.0) | `feat: add dark mode` |
| `fix` | Bug fix | Patch (1.0.1) | `fix: crash on startup` |
| `BREAKING` | Breaking change | Major (2.0.0) | `feat!: redesign UI` |
| `chore` | Maintenance | No release | `chore: update dependencies` |
| `docs` | Documentation | No release | `docs: update README` |
| `style` | Formatting | No release | `style: fix indentation` |
| `refactor` | Code restructuring | No release | `refactor: simplify logic` |
| `test` | Tests | No release | `test: add unit tests` |

### Examples

```
feat: add Linux package support
fix: resolve memory leak in watcher
feat!: change configuration format (BREAKING)
docs: update installation instructions
```

## Release Assets

Each release includes:
- `sentinel.deb` - Debian/Ubuntu package
- `sentinel.rpm` - Fedora/RHEL package  
- `sentinel.pkg.tar.zst` - Arch Linux package

## Troubleshooting

### Release didn't trigger
- Check commit message follows Conventional Commits format
- Only `feat:`, `fix:`, and breaking changes trigger releases
- Other types (`chore:`, `docs:`, etc.) don't trigger releases

### Wrong version bump
- Review commit messages since last release
- `BREAKING CHANGE` in body triggers major version
- `feat:` triggers minor version
- `fix:` triggers patch version


## Configuration

- `.releaserc.json` - Semantic release configuration
- `.github/workflows/ci-cd.yml` - GitHub Actions workflow
- `build/linux/Taskfile.yml` - Linux build tasks
