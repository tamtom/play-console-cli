# Configuration

## Config file

Global: `~/.gplay/config.yaml`
Local (takes precedence): `./.gplay/config.yaml`

```yaml
default_package: com.example.app
timeout: 120s
upload_timeout: 5m
max_retries: 3
debug: false
```

Bootstrap a local config with:

```bash
gplay init
gplay init --package com.example.app --service-account /path/to/sa.json
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `GPLAY_SERVICE_ACCOUNT` | Path to service account JSON |
| `GPLAY_PACKAGE` | Default package name |
| `GPLAY_PROFILE` | Active profile name |
| `GPLAY_TIMEOUT` | Request timeout (e.g., `90s`, `2m`) |
| `GPLAY_TIMEOUT_SECONDS` | Timeout in seconds (alternative) |
| `GPLAY_UPLOAD_TIMEOUT` | Upload timeout (e.g., `5m`, `10m`) |
| `GPLAY_NO_UPDATE` | Disable update checks (set to `1`) |
| `GPLAY_NO_STAR_PROMPT` | Suppress the one-time GitHub star suggestion (set to `1`) |
| `GPLAY_DEBUG` | Enable debug logging (`1` or `api`) |
| `GPLAY_MAX_RETRIES` | Max retries for failed requests (default: 3) |
| `GPLAY_RETRY_DELAY` | Base delay between retries (default: `1s`) |
| `GPLAY_DEFAULT_OUTPUT` | Default output format (`json`, `table`, `markdown`) |
| `GPLAY_AUDIT` | Set to `0` to disable the local audit log |
| `GPLAY_AUDIT_LOG` | Override audit log path (default `~/.gplay/audit.log`) |
| `GPLAY_CHECKS_ACCOUNT` | Default Google Checks account ID |

## Output formats

After a successful `edits commit`, `release`, or `publish track`, gplay prints
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

**JSON-first output.** All commands output minified JSON by default:

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
