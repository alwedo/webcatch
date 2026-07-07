# Release Guide

This project uses **release-please** for automated semantic versioning and releases.

## How It Works

1. **Merge to main** with conventional commit messages
2. **Release-please bot** creates/updates a "Release PR" automatically
3. **Review the Release PR** to see the version bump and changelog
4. **Merge the Release PR** to trigger the actual release

## First Time Setup

### 1. Push the workflow file to GitHub

```bash
git add .github/workflows/release-please.yml
git add release-please-config.json
git add .release-please-manifest.json
git add CHANGELOG.md
git commit -m "ci: add release-please workflow"
git push origin main
```

### 2. Wait for the Release PR

After pushing to main, release-please will:
- Create a "Release PR" (usually titled "chore(main): release 0.1.0")
- This PR shows what will be released

### 3. Merge the Release PR

When you merge it:
- Git tag `v0.1.0` is created
- GitHub release is published
- Binaries are built and attached
- CHANGELOG.md is updated

## Daily Workflow

### Making Changes

Use conventional commit messages:

```bash
# New feature
git commit -m "feat: add webhook filtering"

# Bug fix
git commit -m "fix: resolve template crash on empty endpoint"

# Breaking change
git commit -m "feat!: change API endpoint structure"

# No release
git commit -m "docs: update installation guide"
git commit -m "chore: update dependencies"
```

### Creating a Release

1. **Merge PRs to main** - The Release PR updates automatically
2. **Check the Release PR** - See what version and changes will be released
3. **Merge the Release PR** - Release happens automatically

## Version Bumping Rules

Based on commit prefixes:

- `feat:` → **Minor** (0.1.0 → 0.2.0)
- `fix:` → **Patch** (0.2.0 → 0.2.1)
- `feat!:` or `BREAKING CHANGE:` → **Major** (0.2.1 → 1.0.0)
- `docs:`, `chore:`, `ci:` → **No release**

## What Gets Released

When you merge the Release PR:

1. **Git tag** created (e.g., `v0.1.0`)
2. **GitHub release** published with changelog
3. **Binaries** built for:
   - Linux (amd64)
   - macOS (Intel + Apple Silicon)
   - Windows (amd64)
4. **CHANGELOG.md** updated in the repo

## Tips

- **Batch changes**: Keep the Release PR open and merge multiple PRs. The Release PR updates automatically
- **Review before release**: The Release PR shows exactly what will be released
- **No manual version bumps**: Never edit version numbers manually
- **PR titles matter**: If using "squash and merge", the PR title becomes the commit message

## Troubleshooting

**No Release PR appearing?**
- Check that you pushed to `main` branch
- Check GitHub Actions tab for errors
- Ensure commits use conventional format

**Release PR not updating?**
- It updates on every push to main
- Check the PR's "Files changed" to see what will be released

**Want to skip a release?**
- Just don't merge the Release PR
- Keep working and it will update with new changes

**Need to force a version?**
- Edit `.release-please-manifest.json` to set the version
- Push to main
- Release PR will update
