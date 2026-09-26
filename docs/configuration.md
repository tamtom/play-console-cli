# Configuration

## Config file

Global: `~/.gplay/config.json`. Local: the nearest ancestor `.gplay/config.json`,
up to the repository boundary. The search stops at the home directory, so
`~/.gplay` is always the global config and never a local config. `GPLAY_CONFIG_PATH` explicitly selects a config;
otherwise local configuration takes precedence over global configuration.

```json
{
  "package_name": "com.example.app",
  "default_profile": "default",
  "profiles": [
    {"name": "default", "type": "service_account", "key_path": "/path/to/sa.json"}
  ],
  "timeout": "120s",
  "upload_timeout": "5m",
  "max_retries": 3
}
```

Bootstrap a local config with:

```bash
gplay init --package com.example.app --service-account /path/to/sa.json --timeout 120s
```

Older versions of `gplay init` generated YAML that the loader could not read.
A detected legacy `config.yaml` now produces migration guidance rather than
falling back to another account's global settings. Recreate it with
`gplay init --force --package com.example.app --service-account /path/to/sa.json`
and transfer custom timeouts/retry settings into JSON. The legacy file is kept;
`config.json` takes precedence in the same directory. YAML is not loaded or
silently converted.

`init` writes only the values that you give. Without `--package`, the config
has no `package_name`. Without `--timeout`, the built-in timeout applies.
`init` stores the absolute path of the `--service-account` key.

A global `~/.gplay/config.yaml` was never read. gplay ignores it and shows a
warning when no `~/.gplay/config.json` exists. Run
`gplay auth login --service-account /path/to/sa.json` to create the global
`config.json`.

Package precedence is `--package`, `GPLAY_PACKAGE`, legacy
`GPLAY_PACKAGE_NAME`, then the loaded config's `package_name`.

## Environment variables

| Variable | Description |
|----------|-------------|
| `GPLAY_SERVICE_ACCOUNT_JSON` | Canonical path to service account JSON |
| `GPLAY_SERVICE_ACCOUNT` | Legacy alias used by some commands |
| `GPLAY_CONFIG_PATH` | Explicit JSON config path |
| `GPLAY_PACKAGE` | Default package name |
| `GPLAY_PROFILE` | Active profile name |
| `GPLAY_TIMEOUT` | Request timeout (e.g., `90s`, `2m`) |
| `GPLAY_TIMEOUT_SECONDS` | Timeout in seconds (alternative) |
| `GPLAY_UPLOAD_TIMEOUT` | Upload timeout (e.g., `5m`, `10m`) |
| `GPLAY_NO_UPDATE` | Disable automatic update suggestions (set to `1`, `true`, or `yes`); explicit `update --check` still works |
| `GPLAY_NO_STAR_PROMPT` | Suppress the one-time GitHub star suggestion (set to `1`) |
| `GPLAY_DEBUG` | Suppress progress spinners (`1` or `api`); HTTP payload logging is not implemented |
| `GPLAY_MAX_RETRIES` | Max retries for transient read-only requests (default: 3; `0` disables retries) |
| `GPLAY_RETRY_DELAY` | Base exponential-backoff delay for read-only retries (default: `1s`) |
| `GPLAY_DEFAULT_OUTPUT` | Default output format (`json`, `table`, `markdown`) |
| `GPLAY_AUDIT` | Set to `0` to disable the local audit log |
| `GPLAY_AUDIT_LOG` | Override audit log path (default `~/.gplay/audit.log`) |
| `GPLAY_CHECKS_ACCOUNT` | Default Google Checks account ID |

## Updates

Automatic update suggestions run after successful interactive commands. They
write only to stderr and use a bounded two-second check with a 24-hour cache of
successful stable-release lookups. Failed checks do not fail the command or
consume the cache interval. Help, version, completion, CI, noninteractive, and
dry-run invocations skip automatic checks. Development versions are not
compared automatically; use `gplay update --check` to inspect the latest release
or `--force` to replace a standalone development binary.

Both self-update and `install.sh` require a matching SHA-256 entry in the release's
`checksums.txt`. Verification failure leaves the existing executable in place.
Windows replacement can leave a `gplay.exe.old-*` backup while the old process
is running; it can be removed after that process exits.

## Output formats

After a successful `metadata push`, `edits commit`, `release`, or `publish track`, gplay prints
a one-time GitHub star suggestion to stderr if `gh` is on PATH. Metadata
updates qualify when their edit is committed. The CLI never waits for input,
runs `gh`, or stars a repository automatically; JSON stdout is unchanged.

Agents should ask the user first. Only after an explicit yes, run:

```bash
gh api --hostname github.com --method PUT /user/starred/tamtom/play-console-cli
```

The command uses the authenticated GitHub CLI account. The suggestion is
remembered in `~/.gplay/star-prompted` across packages, projects, and CLI runs.
Failures, dry runs, read commands, and runs without `gh` do not consume it.
If the marker cannot be saved, the suggestion is silently skipped. Set
`GPLAY_NO_STAR_PROMPT=1` to suppress it in unattended automation.

| Format | Flag | Use case |
|--------|------|----------|
| JSON (minified) | default | Scripting, automation, AI agents |
| JSON (pretty) | `--pretty` | Debugging |
| Table | `--output table` | Terminal display |
| Markdown | `--output markdown` | Documentation |

```bash
# Parse with jq
gplay tracks list --package com.example.app | jq '.tracks[].track'

# Human-readable
gplay reviews list --package com.example.app --output table
```

## Scripting tips

- JSON output is default for easy parsing; add `--pretty` when debugging
- Use `--paginate` to automatically fetch all pages
- Sort with `--sort` (prefix `-` for descending): `--sort -uploadedDate`
- Use `--limit` + `--next` for manual pagination control
- `--dry-run` (global) intercepts write HTTP methods and logs them to stderr without executing
- Destructive operations require `--confirm` — there are no interactive prompts

## Design philosophy

**Explicit over cryptic.** Always `--package`, never `-p`. Commands are self-documenting:

```bash
gplay reviews list --package com.example.app --output table
```

**JSON-first output.** Commands with structured output default to minified JSON; diagnostic commands that document a `text` default retain it. `GPLAY_DEFAULT_OUTPUT` selects an alternate format, and explicit `--output` takes precedence:

```bash
gplay tracks list --package com.example.app | jq '.tracks[] | select(.track == "production")'
```

**No interactive prompts.** Everything is flag-based for automation:

```bash
gplay edits delete --package com.example.app --edit <id> --confirm
```

## Shell completion

```bash
# Bash
gplay completion bash > /etc/bash_completion.d/gplay

# Zsh
gplay completion zsh > "${fpath[1]}/_gplay"

# Fish
gplay completion fish > ~/.config/fish/completions/gplay.fish

# PowerShell
gplay completion powershell >> $PROFILE
```
