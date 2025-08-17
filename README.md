<p align="center">
    <picture>
        <img src="https://github.com/flowmitry/syncai/raw/main/doc/assets/syncai_github.png" width="194">
    </picture>
</p>

---

**SyncAI** is a lightweight utility that keeps AI-assistant guidelines, rules and ignored files in sync across multiple agents:

* Cursor
* GitHub Copilot
* JetBrains Junie
* Cline

It watches the files you specify in a JSON configuration and propagates every change to the corresponding locations for the other agents.

## Quick start

1. Download a suitable binary from the [GitHub Releases](https://github.com/flowmitry/syncai/releases)
2. Copy [syncai.json](syncai.json) to your project
3. Launch the binary in the project dir or with an argument `./syncai -config {path_to_syncai.json}`


To build SyncAI manually, follow the next steps:

```bash
cd syncai

# Download dependencies
go mod tidy

# Run with default config path (syncai.json)
go run .
```

## Configuration format

The default configuration is a simple JSON map:

```json
{
  "config": {
    "interval": 5
  },
  "agents": [
    {
      "name": "cursor",
      "rules": {
        "pattern": ".cursor/rules/*.mdc"
      },
      "guidelines": {
        "path": ".cursorrules"
      },
      "ignore": {
        "path": ".cursorignore"
      }
    },
    {
      "name": "copilot",
      "rules": {
        "pattern": ".github/instructions/*.instruction.md"
      },
      "guidelines": {
        "path": ".github/guidelines.md"
      }
    },
    {
      "name": "cline",
      "rules": {
        "pattern": ".clinerules/*.md"
      }
    },
    {
      "name": "junie",
      "guidelines": {
        "path": ".junie/guidelines.md"
      },
      "ignore": {
        "path": ".aiignore"
      }
    }
  ]
}
```

## How it works

1. SyncAI loads the configuration file and builds a watch-list of directories and files derived from all sections.
2. It periodically scans those files for new or updated modification times.
3. When a rule file changes, its contents are copied to every other agent’s rule directory.

The copying logic is intentionally simple and conservative:

* The filename is preserved exactly, unless the target pattern contains a `*` wildcard—in that case, the wildcard is replaced with the source file’s base name.
* Destination directories are created as needed.

## How to update

SyncAI implements self-update. Use the following command:

```bash
syncai -self-update
```
