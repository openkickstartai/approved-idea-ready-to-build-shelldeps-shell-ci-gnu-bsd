# ShellDeps

Shell script external command dependency scanner and cross-platform portability guard.

Catches undeclared binary dependencies and GNU/BSD incompatibilities **before** your CI explodes.

## Features

- 📦 Detects every external command dependency in shell scripts
- ⚠️ Flags GNU/BSD incompatible flags (`sed -i`, `grep -P`, `stat -c`, etc.)
- 🔍 Parses pipe chains, `&&`/`||` chains, command substitutions
- 📋 JSON output for CI integration
- 🚀 Single static binary — zero runtime dependencies

## Install

```bash
go install github.com/shelldeps/shelldeps@latest
```

Or build from source:

```bash
git clone https://github.com/shelldeps/shelldeps.git && cd shelldeps
go build -o shelldeps .
```

## Usage

```bash
# Scan a single script
shelldeps deploy.sh

# Scan a directory recursively
shelldeps ./scripts/

# JSON output
shelldeps --format json deploy.sh

# CI gate mode — exit 1 if any finding
shelldeps --check deploy.sh

# Only show GNU/BSD compat warnings
shelldeps --compat-only deploy.sh
```

### Example Output

```
📦 deploy.sh:3  [curl] external command: curl
📦 deploy.sh:4  [jq] external command: jq
⚠️  deploy.sh:7  [sed] sed -i without '' is GNU-only; BSD needs sed -i ''
⚠️  deploy.sh:9  [grep] grep -P (PCRE) is GNU-only; use -E instead
```

## GitHub Actions

```yaml
- name: Shell portability check
  run: |
    go install github.com/shelldeps/shelldeps@latest
    shelldeps --check ./scripts/
```

## License

MIT
