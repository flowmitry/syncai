<p align="center">
    <picture>
        <img src="https://github.com/flowmitry/syncai/raw/main/doc/assets/syncai_github.png" width="194">
    </picture>
</p>

---

**SyncAI** is a lightweight utility that keeps AI-assistant guidelines, rules and ignored files in sync across multiple
agents:

* Cursor
* GitHub Copilot
* JetBrains Junie
* Cline
* Claude Code

It watches the files you specify in a JSON configuration and propagates every change to the corresponding locations for
the other agents.

## Quick start

1. Download a suitable binary from the [GitHub Releases](https://github.com/flowmitry/syncai/releases)
2. Copy [syncai.json](syncai.json) to your project and leave configuration for your agents
3. Launch the binary in the project dir or with an argument `./syncai -config {path_to_syncai.json}`

To build SyncAI manually, follow the next steps:

```bash
cd syncai

make build
```

## Configuration format

The default configuration is a simple JSON map (for more details check [syncai.json](syncai.json)):

```json
{
  "config": {
    // sync interval in seconds
    "interval": 5
  },
  "agents": [
    {
      // agent name
      "name": "<AGENT_NAME>",
      // optional "rules" section
      // GitHub Copilot calls it "instructions", Cursor and Cline "rules"
      "rules": {
        "pattern": ".<AGENT>/rules/*.md"
      },
      // optional "context" section
      "context": {
        "path": "/path/to/your/guidelines.md"
      },
      // optional "ignore" section
      "ignore": {
        "path": "/path/to/your/ignorefile"
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

* The filename is preserved exactly, unless the target pattern contains a `*` wildcard—in that case, the wildcard is
  replaced with the source file’s base name.
* Destination directories are created as needed.

## How to update

SyncAI implements self-update. Use the following command:

```bash
syncai -self-update
```

## How to integrate it with CI

If you want to run the initial sync only (for CI, for example), use `./syncai -no-watch`