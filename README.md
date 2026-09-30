# Pi Agent Lab

A framework for creating, versioning, and comparing [Pi](https://github.com/earendil-works/pi) agents against standardized tasks. Run it from a Go CLI, or from a [Herdr](https://herdr.dev) plugin.

You define an **agent** (a name and a purpose), give it one or more immutable **versions** (a specific model, system prompt, tool policy, extensions and skills), define a **task** (a standardized prompt you'll run every version against), and run them. Every run is recorded automatically and can be rated on three axes:

- Did it accomplish the task
- How much rework did it need
- How much intervention did it need

This allows you to actually compare "code-reviewer v1" against "code-reviewer v2" on the same input instead of trusting a vague impression.

## Structure

| Piece                                      | What it is                                                                                                                                                                                                                                                                        |
| ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`pi-session-record`](pi-session-record)   | A Pi extension (TypeScript/Node) that writes one JSON record per session to `~/.pi/agent/records/<sessionId>.json`. Usage stats, tool activity, and an optional `/rate` outcome. It reads `PI_AGENT_VERSION`/`PI_TASK_ID` from the environment so records can be grouped by task. |
| [`pi-session-schema`](pi-session-schema)   | The JSON Schema for a session record. Everything else derives its type from this.                                                                                                                                                                                                 |
| [`pi-agent-registry`](pi-agent-registry)   | Reads/writes the on-disk registry of agents, versions, and tasks (plain JSON + Markdown files, git-friendly).                                                                                                                                                                     |
| [`pi-agent-validate`](pi-agent-validate)   | Validates model IDs (via `pi --list-models`), tool names, extension/skill source strings, and thinking levels.                                                                                                                                                                    |
| [`pi-agent-invoke`](pi-agent-invoke)       | Builds the exact `pi ...` shell command for a given agent version + task, or for resuming a past session.                                                                                                                                                                         |
| [`pi-agent-compare`](pi-agent-compare)     | Groups recorded runs by task and ranks the rated ones (outcome, then rework, then autonomy; all higher-is-better, cost only breaks a full tie).                                                                                                                                   |
| [`pi-agent-lab`](pi-agent-lab)             | The CLI. Everything above, wired into CRUD commands.                                                                                                                                                                                                                              |
| [`herdr-pi-agent-lab`](herdr-pi-agent-lab) | A Herdr plugin: Bubble Tea TUI panes that drive the same registry and comparison logic.                                                                                                                                                                                           |

The CLI and the TUI are two interfaces over the same on-disk registry.

## Prerequisites

- [Pi](https://github.com/earendil-works/pi) installed and on `PATH` (`pi --list-models` must work — it's how model IDs get validated).
- Go 1.26+ to build the CLI and the TUI plugin.
- [Herdr](https://herdr.dev) if you want the TUI.

## Installation

```sh
# CLI
cd pi-agent-lab/
./scripts/install.sh

# TUI plugin (registers the plugin into your Herdr installation after building)
cd herdr-pi-agent-lab/
./scripts/install.sh
```

## Registry setup

Before anything else, tell `pi-agent-lab` where to keep its data:

```sh
pi-agent-lab config default          # use the suggested location, after a y/n confirmation
# or
pi-agent-lab config update /path/to/registry
pi-agent-lab config show             # print the resolved path
```

Or use the Config section in the Herdr plugin.

This is stored in `$XDG_CONFIG_HOME/pi-agent-lab/config.json` (default `~/.config/pi-agent-lab/config.json`) and points at a registry directory (suggested default: `$XDG_DATA_HOME/pi-agent-lab/registry`, i.e. `~/.local/share/pi-agent-lab/registry`). **Nothing else works until this is set**. Every command that needs it will tell you so and name the exact fix.

**Registry objects are immutable.** There is no `update` command for an agent, version, or task. Once created, the only way to change one is to delete it and recreate it. This keeps a version's identity meaningful: if you could edit `code-reviewer@2.0.0` in place, two runs tagged with that same version ID could have silently been produced by completely different agents.

## Usage

For the CLI usage instructions run:

```sh
pi-agent-lab --help
```

The CLI has autocompletions on most completable commands/flags/values...

The TUI has hints for usage built into the UI.

See the sections below for common usage scenarios.

## Quick start (CLI)

```sh
pi-agent-lab create agent code-reviewer \
  --name "Code Reviewer" \
  --description "Read-only reviewer for diffs and PRs" \
  --initial-version-id 1.0.0 \
  --model claude-sonnet-5 --thinking medium \
  --system-prompt-file ./reviewer-prompt.md \
  --tools-mode allowlist --tools read,grep,find,ls

pi-agent-lab create task review-auth-pr \
  --name "Review auth PR" \
  --prompt-file ./review-auth-pr.md

pi-agent-lab list agents
pi-agent-lab show agent-version code-reviewer@1.0.0

pi-agent-lab invocation code-reviewer@1.0.0 review-auth-pr
# prints the exact `pi ...` command it would run, asks
#   run this command? [y/N]
# and only executes on y
```

A version's identity is everything that affects agent _behavior_: model, provider, thinking level, system prompt (and whether it replaces or appends to Pi's default one), the tool policy, and the extensions/skills it loads. Two versions with an identical system prompt but a different model, or a different tool policy, are genuinely different agents. That's why they get separate version IDs rather than being folded into one.

### Flags

#### Tool policy

`--tools-mode`:

- `all` (default) - every built-in tool, no `--tools`/`--exclude-tools` needed.
- `none` - `--no-tools`.
- `extensions-only` - `--no-builtin-tools`; only tools an extension provides.
- `allowlist` / `denylist` - `--tools`/`--exclude-tools <comma-separated names>`.

#### Extensions and skills

- `--extensions-isolate` / `--skills-isolate` - don't load extensions/skills from your ambient Pi setup for this version, only what's explicitly listed still loads. Without isolation, a version's behavior can silently drift if you install or remove global Pi extensions/skills between two runs of "the same" version.
- `--extension` (repeatable) - a package reference (`npm:name@1.2.3` or `git:host/repo@ref`, pinned to an exact version), a path relative to the registry directory, or an absolute/`~` path (accepted, with a warning that it might only work on the calling machine).
- `--skill` (repeatable) - a path to a skill folder (containing `SKILL.md`) or a single `.md` file with skill frontmatter. It's **copied into the registry** under the version, so the version stays portable even if the original skill file later moves or is deleted.

**Caveat on isolation and recording**: `pi-session-record` is just another Pi extension. If a version sets `--extensions-isolate` and doesn't also list `pi-session-record` in its own `--extension` values, that version's runs won't get recorded. There's no automatic safety net that re-adds it. Add it explicitly (e.g. `--extension ../pi-session-record` relative to the registry, or wherever it actually lives) on any isolated version you still want a record from.

#### Running and recording

`pi-agent-lab invocation <agent>@<version> <task>` (and the TUI's "Start a run") build a command shaped like:

```sh
id=$(uuidgen) && PI_AGENT_VERSION='code-reviewer@1.0.0' PI_TASK_ID='review-auth-pr' pi --session-id "$id" \
  --model 'claude-sonnet-5' --provider 'opencode' --thinking 'medium' \
  --tools 'read,grep,find,ls' \
  --system-prompt '...' '@/path/to/registry/tasks/review-auth-pr.prompt.md'
```

`PI_AGENT_VERSION` and `PI_TASK_ID` are what let `pi-session-record` tag the resulting record so it can be grouped and compared later. This is the whole mechanism, there's no separate "run tracking" database.

#### Rating a run

Inside the Pi session, run `/rate`. You'll be asked three 1-5 questions, all **higher is better**:

1. Did it accomplish the task? (1 no - 5 fully)
2. Rework needed? (1 lots - 5 none)
3. Intervention needed? (1 constant - 5 none)

An optional note can follow. A session with no rating is never scored. It shows up in a separate "unrated" list, not blended in with some placeholder value.

#### Comparing runs

```sh
pi-agent-lab list runs --task review-auth-pr
```

Rated runs for a task are ranked by **outcome, then rework, then autonomy** (lexicographically, all higher-is-better), never collapsed into one weighted score, since deciding how to weight them isn't this tool's call to make. Runs still tied on all three are then differentiated only by cost as a last resort. Token cost, total tokens, tool errors, and duration are always shown as independent stats - a cheaper run that hit more tool errors isn't automatically "better."

## Other CLI commands

```
pi-agent-lab create   agent | agent-version | task
pi-agent-lab show     agent | agent-version | task
pi-agent-lab list     agents | agent-versions | tasks | runs
pi-agent-lab delete   agent | agent-version | task
pi-agent-lab invocation <agent>@<version> <task>
pi-agent-lab config    default | update <dir> | show
```

- `show agent-version` keeps its output compact by default (extensions/skills as name lists only) - pass `--skill <name>` (repeatable) to also print one skill's full rendered content. `show agent-version`/`show task` render the system prompt / task prompt as real terminal markdown (via [glamour](https://github.com/charmbracelet/glamour)), not raw `#`/`*` characters.
- `delete agent` removes an agent and all its versions; there is no per-field edit anywhere. See "Registry objects are immutable" above.
- Shell completion is available via `pi-agent-lab completion <shell>`; agent/version/task IDs, model names, tool names, thinking levels, and (for `show agent-version --skill`) the version's actual skill names all complete.

## The TUI (Herdr plugin)

Once built and linked (see "Installation" above), open it from Herdr:

```sh
herdr plugin pane open --plugin darkokos.pi-agent-lab --entrypoint menu
```

Five panes, each its own binary under `bin/`:

| Pane           | Entrypoint | What it's for                                                                                                                                                                                                    |
| -------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Menu           | `menu`     | Launches the other four panes.                                                                                                                                                                                   |
| Agents & Tasks | `registry` | Create/delete agents, versions, and tasks; browse an agent version's full details (system prompt, tools, extensions, skills - each skill's content expandable on demand so a long one doesn't flood the screen). |
| Start a run    | `run`      | Pick an agent version and a task, review the exact command, confirm, run.                                                                                                                                        |
| Review Runs    | `compare`  | The ranked comparison from `list runs`, plus resuming an unrated session straight into `/rate`.                                                                                                                  |
| Config         | `config`   | Same as `config update`/`config default`/`config show`, interactively - the default-location action asks for `y`/`n` confirmation before writing anything.                                                       |

### Keybindings

Universal, on every screen in every pane:

- `ctrl+u` / `ctrl+d` - half-page scroll up/down
- `q` / `ctrl+c` - quit the pane

Per-screen, shown live in that screen's own pinned footer:

- `tab` to switch sub-tabs
- `j`/`k`/arrows to move a selection
- `enter` to open/pick/toggle
- `n` for new
- `d` for delete (always behind a `y`/`n` confirmation screen)
- `r` to run or resume a run
- `esc`/`backspace` to go back
