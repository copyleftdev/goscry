# Changelog

All notable changes to GoScry will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- CI/CD pipeline with GitHub Actions for automated builds and releases
- Cross-platform binary releases for Linux, macOS (Intel and ARM), and Windows
- Documentation for release process

### Fixed
- Integration tests now properly handle task completion and status transitions
- Fixed format string issues in server error handlers

## [0.1.0] - 2025-03-28

### Added
- Initial release of GoScry
- Task-based API for browser control
- Support for navigation, clicking, typing, and DOM extraction
- 2FA detection and handling capability
- Configurable server with YAML or environment variables
- Comprehensive testing framework

[Unreleased]: https://github.com/copyleftdev/goscry/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/copyleftdev/goscry/releases/tag/v0.1.0
