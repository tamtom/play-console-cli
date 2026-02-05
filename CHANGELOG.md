# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-02-05

### Added
- Browser-based OAuth login - `gplay auth login` now opens your browser for authentication
- `Agents.md` documentation for AI coding agents
- Agent Skills section in README with link to [gplay-cli-skills](https://github.com/tamtom/gplay-cli-skills)

### Changed
- `gplay auth login` defaults to browser OAuth flow
- Service account auth moved to `gplay auth login --service-account <path>`
- `CLAUDE.md` now references `Agents.md`

## [0.1.0] - 2025-02-05

### Added
- Initial release
- Authentication: service account and OAuth profiles
- Edit lifecycle: create, get, validate, commit, delete
- Bundle and APK upload/list
- Track management: list, get, update, patch
- Store listings: CRUD operations by locale
- Images: upload, delete, list by type
- Reviews: list, get, reply
- High-level commands: release, promote, rollout
- Monetization: in-app products, subscriptions, base plans, offers
- Purchase management: orders, product/subscription verification
- FastLane integration: sync, import/export, validate
- Shell completion: bash, zsh, fish, powershell
- Self-update mechanism
- Homebrew tap distribution
