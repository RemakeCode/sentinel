# Automated Release Guide

This project uses [semantic-release](https://semantic-release.gitbook.io/) with [Conventional Commits](https://www.conventionalcommits.org/) to automate versioning and releases.

## How It Works

- Every push to `main` branch triggers the release workflow
- Commit messages are analyzed to determine version bumps
- Linux packages (AppImage, .deb, .rpm) are built automatically
- GitHub Release is created with changelog
- Version tags are created automatically

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

## Branch Strategy

### `main` branch
- Production releases (e.g., `v1.0.0`, `v1.1.0`, `v1.0.1`)
- Stable releases only

### `beta` branch (optional)
- Pre-releases (e.g., `v1.0.0-beta.1`, `v1.0.0-beta.2`)
- For testing before production

### `alpha` branch (optional)
- Early pre-releases (e.g., `v1.0.0-alpha.1`)
- For experimental features

## Release Assets

Each release includes:
- `sentinel-{version}.x86_64.AppImage` - Universal Linux binary
- `sentinel_{version}_amd64.deb` - Debian/Ubuntu package
- `sentinel-{version}.x86_64.rpm` - Fedora/RHEL package

## Manual Override

To trigger a specific version bump, use commit message footers:

```
fix: minor update

[skip ci]

Release Notes: This is a manual version bump
```

## First Release

The initial release will be `v1.0.0`. To start from `v0.x.x`, the first commit should be:

```
chore: initial release

Release Notes: Initial development release
```

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

### Build failed
- Check GitHub Actions logs
- Verify all dependencies are available
- Ensure Task and Wails CLI install correctly

## Local Testing

Test your commit messages locally:

```bash
# Check what version would be released
npx semantic-release --dry-run

# Verify commit message format
npx commitlint --from HEAD~1
```

## Configuration

- `.releaserc.json` - Semantic release configuration
- `.github/workflows/ci-cd.yml` - GitHub Actions workflow
- `build/linux/Taskfile.yml` - Linux build tasks
