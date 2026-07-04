# winget packaging

Manifests for publishing `gplay` to the [Windows Package Manager](https://learn.microsoft.com/windows/package-manager/) community repository, so users can install with:

```powershell
winget install tamtom.gplay
```

## How it works

`gplay` ships as a single portable binary. The manifest uses `InstallerType: zip`
with a nested `portable` installer and `PortableCommandAlias: gplay`, so winget
downloads `gplay-windows-amd64.zip` (which contains `gplay.exe`), extracts it, and
exposes a `gplay` command on PATH.

The release workflow (`.github/workflows/release.yml`) builds that zip for every
tagged release and includes it in the GitHub release assets.

## Initial submission (one-time)

The package must exist in `microsoft/winget-pkgs` before automated updates can run.

1. Fork <https://github.com/microsoft/winget-pkgs>.
2. Copy `manifests/t/tamtom/gplay/0.6.0/` into the fork under the same path.
3. Open a PR to `microsoft/winget-pkgs`. The validation bot verifies the installer
   URL and SHA-256, then a moderator merges it (usually 1–2 days).

Alternatively, from a Windows machine:

```powershell
winget install wingetcreate
wingetcreate submit --token <PAT> winget/manifests/t/tamtom/gplay/0.6.0
```

## Automated updates (subsequent releases)

The `winget` job in the release workflow runs `wingetcreate update` on each tagged
release and submits a version-bump PR automatically. It requires a repository
secret **`WINGET_TOKEN`** — a classic GitHub PAT (public_repo scope) on an account
that has forked `microsoft/winget-pkgs`. Without the secret the job logs a skip and
succeeds, so releases are never blocked.
