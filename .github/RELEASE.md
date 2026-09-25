# Automated Release Guide

This project uses [semantic-release](https://semantic-release.gitbook.io/) with [Conventional Commits](https://www.conventionalcommits.org/) to choose the version and generate GitHub Release notes. Stable releases are triggered manually from `main` and created as drafts so the notes can be reviewed and edited before publishing.

## How It Works

- Pushes to `main` run tests and build checks
- A stable GitHub Release is created manually through the workflow's **Run workflow** button on `main`
- Commit messages are analyzed to determine version bumps
- Linux packages (`.deb`, `.rpm`, `.pkg.tar.zst`, and `.flatpak`) and the Decky plugin ZIP are built automatically
- Generated release notes are attached to a draft GitHub Release; edit them in GitHub before publishing
- Release notes are kept on GitHub Releases; semantic-release does not generate or commit a repository changelog
- Version tags are created automatically

## Workflow Triggers

| Event | Jobs Run | Description |
|-------|----------|-------------|
| Push to `main` | test, build-check | Automatic CI on every commit |
| Manual Dispatch from `main` | test, release | Builds assets and creates a draft release |

### Manual Release

To create a release:
1. Go to GitHub Actions → CI/CD workflow
2. Click "Run workflow"
3. Select `main` and click "Run workflow"

The release job will:
- Run tests first
- Analyze commit messages since last release
- Determine the next version from Conventional Commits
- Build and upload the Linux packages and Decky ZIP
- Create a draft GitHub Release with generated notes
- Review/edit the draft notes and publish the release in GitHub

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
| `feat` | New feature | Minor | `feat: add dark mode` |
| `fix` | Bug fix | Patch | `fix: crash on startup` |
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

Each stable Linux release includes:
- `sentinel-<version>.deb` - Debian/Ubuntu package
- `sentinel-<version>.rpm` - Fedora/RHEL package
- `sentinel-<version>.pkg.tar.zst` - Arch Linux package
- `sentinel-decky-plugin-<version>.zip` - Decky plugin
- `sentinel-<version>-x86_64.flatpak` - Flatpak bundle

The project currently publishes Linux desktop builds and the Steam Deck Decky plugin. It does not publish Windows or macOS app builds.

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

- `.releaserc.json` - Semantic-release configuration
- `.github/workflows/ci-cd.yml` - GitHub Actions workflow
- `build/linux/Taskfile.yml` - Linux build tasks
