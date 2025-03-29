# Release Process for GoScry

This document outlines the process for creating new releases of GoScry.

## Automated Release Process via GitHub Actions

GoScry uses GitHub Actions for continuous integration and deployment. The CI/CD pipeline automatically builds, tests, and creates releases when tags are pushed.

### How to Create a New Release

1. Ensure all changes are committed and pushed to the main branch
2. Tag the commit with a version number following semantic versioning (e.g., `v1.0.0`)
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
3. The GitHub Actions workflow will automatically:
   - Build and test the code
   - Create binaries for multiple platforms (Linux, macOS, Windows)
   - Create a GitHub Release with the built binaries
   - Generate release notes based on the commits since the last tag

### Release Artifacts

The following binaries are automatically built and attached to each release:

- Linux (AMD64): `goscry-linux-amd64.zip`
- macOS (AMD64): `goscry-darwin-amd64.zip`
- macOS (ARM64): `goscry-darwin-arm64.zip`
- Windows (AMD64): `goscry-windows-amd64.zip`

## Versioning

GoScry follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for functionality added in a backward compatible manner
- **PATCH** version for backward compatible bug fixes

## Release Checklist

Before tagging a new release, ensure:

1. All tests pass locally and in CI
2. Documentation is updated
3. CHANGELOG.md is updated with a summary of changes
4. Version numbers are updated in relevant files

## Hotfix Releases

For urgent fixes:

1. Create a branch from the tag that needs fixing
2. Apply the fix
3. Tag with an incremented patch version
4. Push the tag
