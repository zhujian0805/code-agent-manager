# Metadata Index and Shared Go Search/Install Design

Date: 2026-06-17

## Goal

CAM will add a shared Go metadata layer for agents, skills, and plugins. The layer will ingest repository metadata configured in `~/.config/code-agent-manager/*_repos.json` and `~/.config/code-agent-manager/config.yaml`, store normalized records in SQLite, and expose local-first search, refresh, and install operations to both the Go CLI and desktop sidecar.

Python CLI code is not extended for this feature. Python removal is a later follow-up after the Go implementation is complete.

## Architecture

The feature adds one shared Go metadata subsystem used by both CLI and desktop APIs:

```text
~/.config/code-agent-manager/config.yaml
~/.config/code-agent-manager/*_repos.json
             │
             ▼
   Go metadata refresh service
             │
             ▼
   SQLite metadata index
             │
      ┌──────┴──────┐
      ▼             ▼
 Go CLI        Desktop sidecar API
```

The metadata subsystem is responsible for loading source configuration, refreshing item metadata, storing normalized records, searching SQLite first, and installing selected items into target coding agents through shared Go code.

## Components

### Shared metadata package

A new Go package owns metadata models, refresh logic, search logic, SQLite persistence, and install orchestration. It defines the supported item kinds: `agent`, `skill`, and `plugin`.

The package exposes methods equivalent to:

- `Refresh(ctx, options)`
- `Search(ctx, query, filters)`
- `Get(ctx, id)`
- `Install(ctx, id, targetApp, options)`

### Config/source loader

The loader reads user configuration first and falls back to bundled repository defaults where existing CAM behavior expects it. Inputs include:

- `~/.config/code-agent-manager/config.yaml`
- `~/.config/code-agent-manager/agent_repos.json`
- `~/.config/code-agent-manager/skill_repos.json`
- `~/.config/code-agent-manager/plugin_repos.json`
- bundled fallback repo metadata files

User configuration has priority over bundled defaults.

### SQLite store

The store persists configured sources and normalized items. It runs migrations automatically before metadata operations and uses upserts so refresh can be repeated safely.

### Discovery/refresh adapters

Separate adapters handle each source type because conventions differ:

- agents: markdown files under configured `agentsPath`
- skills: skill directories or markdown files with frontmatter
- plugins: plugin or marketplace metadata files

### CLI integration

The Go CLI gains a metadata command group for the shared workflow:

```bash
cam metadata refresh
cam metadata search <query>
cam metadata search <query> --type agent
cam metadata search <query> --type skill
cam metadata search <query> --type plugin
cam metadata install <item-id> --target claude
```

Domain-specific aliases such as `cam skill search` can be added later, but the first implementation should avoid duplicate behavior.

### Desktop sidecar integration

The sidecar exposes metadata endpoints for refresh, search, and install. The frontend can use these APIs to show local SQLite results first, then offer an explicit refresh action.

## Data Flow

### Refresh

```text
User clicks refresh / runs CLI refresh
        │
        ▼
Load config.yaml + *_repos.json
        │
        ▼
Resolve enabled agent/skill/plugin repos
        │
        ▼
Fetch or read source metadata
        │
        ▼
Parse agents / skills / plugins
        │
        ▼
Upsert normalized records into SQLite
        │
        ▼
Return refresh summary
```

Refresh is explicit. Search stays local and fast by default.

### Search

```text
User searches query
        │
        ▼
Search SQLite first
        │
        ▼
Return local results immediately
        │
        ▼
Optional refresh action
        │
        ▼
Pull configured online/GitHub sources
        │
        ▼
Update SQLite
        │
        ▼
Search again
```

This implements the selected behavior: SQLite first, then refresh.

### Install

```text
User selects item
        │
        ▼
Load item from SQLite by ID
        │
        ▼
Resolve target coding agent
        │
        ▼
Fetch source repo if needed
        │
        ▼
Copy/install item into target agent location/config
        │
        ▼
Update installed status metadata
```

CLI and desktop both use the same Go install path.

## SQLite Schema

### `metadata_sources`

Stores configured repositories and source metadata.

Fields:

- `id`
- `kind`: `agent`, `skill`, or `plugin`
- `source_key`: stable key such as `owner/repo`
- `owner`
- `repo`
- `branch`
- `path`
- `enabled`
- `source_file`
- `last_refreshed_at`
- `created_at`
- `updated_at`

### `metadata_items`

Stores searchable installable records.

Fields:

- `id`
- `kind`: `agent`, `skill`, or `plugin`
- `name`
- `description`
- `source_id`
- `repo_owner`
- `repo_name`
- `repo_branch`
- `item_path`
- `install_key`
- `target_apps`
- `metadata_json`
- `installed`
- `installed_targets`
- `last_seen_at`
- `created_at`
- `updated_at`

### Search strategy

The first implementation uses indexed `LIKE` searches across `name`, `description`, `repo_owner`, `repo_name`, and `install_key`. SQLite FTS can be added later if needed.

## Online/GitHub Behavior

Search is local-only by default. The online switch means refresh from configured remote/GitHub sources, update SQLite, then search SQLite again. It does not mean broad, arbitrary GitHub search in this feature.

This keeps behavior deterministic and aligned with CAM's configured marketplace/repository model.

## Error Handling

Refresh is resilient. One broken repository does not fail the whole refresh. Refresh returns a summary with sources scanned, items indexed, items updated, stale records, failed sources, and warnings.

Install reports clear errors for missing item IDs, unsupported targets, failed downloads, existing installed items, and partial failures. It does not mark an item installed if installation fails.

SQLite migration or database errors stop the command/API request with the database path and actionable context.

## Testing

Unit tests cover:

- config loading priority
- parsing agent, skill, and plugin repo files
- metadata normalization
- SQLite migrations
- SQLite upsert/search behavior
- stale item handling
- install target validation

Integration-style tests use temporary directories for fake CAM config, fake source repositories, a temporary SQLite database, and fake target agent directories. They cover refresh, search, install, and refresh-after-source-change flows.

CLI tests cover the new `cam metadata refresh`, `cam metadata search`, and `cam metadata install` command behavior where practical.

Because the implementation changes Go code, repository test requirements apply after coding.

## Scope

Included:

- Shared Go metadata service
- SQLite metadata index
- Config/repo loading from user and bundled sources
- Local SQLite search
- Explicit refresh from configured local/remote sources
- Shared install path for selected agents, skills, and plugins
- Go CLI metadata commands
- Desktop sidecar metadata APIs

Excluded:

- Deleting Python CLI files
- Full arbitrary GitHub global search
- Replacing every existing domain-specific command
- Major frontend redesign beyond wiring search/refresh/install if supported by the current UI
- New external registry service
