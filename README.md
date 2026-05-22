# env-audit

> Your `.env` files are a mess. Let's find out how bad.

`env-audit` scans every `.env*` file across all your projects and tells you what's wrong:

- **Exposed in git** — `.env` files that are committed and readable by anyone with repo access
- **Duplicate secrets** — the same secret value copy-pasted across multiple projects
- **Unreferenced vars** — defined in `.env` but never used in any source file (dead config)

```
❯ env-audit

  EXPOSED IN GIT  38 files
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  myapp/.env                     12 vars  ← tracked in git!
  old-project/.env.local          5 vars  ← tracked in git!
  ...

  DUPLICATE SECRETS  203 pairs
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  STRIPE_SECRET_KEY  (sk_live_4xV***)
    myapp/.env, reputrack/.env, old-api/.env
  DATABASE_URL  (postgres://us***)
    myapp/.env, staging/.env

  UNREFERENCED VARS  257 vars
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  myapp/.env
    OLD_SENDGRID_KEY    — not found in any source file
    LEGACY_WEBHOOK_URL  — not found in any source file

  SUMMARY
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  88 .env files  ·  1151 vars  ·  38 exposed  ·  203 duplicate secrets  ·  257 unreferenced
```

*(These are real numbers from a real dev machine.)*

## Install

### One line (no Go needed)

```bash
curl -fsSL https://raw.githubusercontent.com/dawitlabs/env-audit/main/install.sh | sh
```

### Go

```bash
go install github.com/dawitlabs/env-audit@latest
```

### Manual — download binary

Go to [Releases](https://github.com/dawitlabs/env-audit/releases/latest):

| Platform | File |
|----------|------|
| Linux x86_64 | `env-audit-linux-amd64` |
| Linux ARM64 | `env-audit-linux-arm64` |
| Mac (Apple Silicon) | `env-audit-darwin-arm64` |
| Mac (Intel) | `env-audit-darwin-amd64` |

```bash
chmod +x env-audit-linux-amd64 && mv env-audit-linux-amd64 ~/.local/bin/env-audit
```

## Usage

```bash
env-audit                        # scan ~/projects (default)
env-audit --root ~/dev           # scan a different directory
env-audit --root .               # scan current project only
```

## What it checks

| Check | What it means |
|-------|---------------|
| Exposed in git | File is tracked by git — anyone who clones the repo can read your secrets |
| Duplicate secrets | Same value in multiple projects — rotating one means rotating all |
| Unreferenced vars | Key never appears in any `.go/.ts/.js/.py` etc. source file — probably dead config |

## How it works

Walks the target directory, finds every `.env*` file (`.env`, `.env.local`, `.env.production`, etc.), parses `KEY=value` pairs, then:

- Runs `git ls-files --error-unmatch` to detect tracked files
- Cross-references key+value pairs across all files for duplicates
- Greps source files by key name to detect unreferenced vars

Zero network calls. Zero dependencies. Single static binary.

## License

MIT
