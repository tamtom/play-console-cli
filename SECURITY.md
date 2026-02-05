# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please DO NOT report security vulnerabilities through public GitHub issues.**

Instead, please report security vulnerabilities by emailing:

**security@tamtom.dev** or open a private security advisory at https://github.com/tamtom/play-console-cli/security/advisories/new

You should receive a response within 48 hours. If for some reason you do not, please follow up to ensure we received your original message.

Please include the following information:

- Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

## Security Update Process

1. **Acknowledgment** - We'll acknowledge receipt within 48 hours
2. **Investigation** - We'll investigate and determine severity (1-7 days)
3. **Fix Development** - We'll develop a fix (depends on severity)
4. **Testing** - We'll test the fix thoroughly
5. **Release** - We'll release a patch version
6. **Disclosure** - We'll publish a security advisory on GitHub

## Security Best Practices

When using `gplay`:

### DO:
- Store service account keys securely (never commit to git)
- Use environment variables or secure secret management in CI/CD
- Limit service account permissions to minimum required
- Rotate service account keys regularly
- Use different service accounts for different environments
- Review audit logs periodically

### DON'T:
- Commit service account JSON files to version control
- Share service account keys via chat/email
- Use production credentials in development
- Grant excessive permissions to service accounts
- Store credentials in plain text config files (store paths only)

## Known Security Considerations

- **Credentials Storage**: Service account file paths (not contents) are stored in config files
- **Temporary Files**: Access tokens are never written to disk, only held in memory
- **Logs**: Debug mode redacts sensitive information (tokens, keys)
- **File Permissions**: Config files are created with 0600 permissions (user read/write only)

## Dependency Security

We use:
- Dependabot for automated dependency updates
- `govulncheck` for Go vulnerability scanning
- `gosec` for static security analysis

All dependencies are reviewed before merging.
