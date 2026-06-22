# MCP Catalog Source Migration Design

Date: 2026-06-22

## Goal

Move CAM's MCP server catalog out of `code-agent-manager` and make `Chat2AnyLLM/awesome-mcp-servers` the source of truth for MCP server records. CAM should continue to manage MCP client configuration and installation, but it should retrieve installable MCP server definitions from configured online/local catalog sources, matching the pattern used for skills, plugins, prompts, and subagents.

## Scope

This work covers only MCP server catalog ownership and retrieval.

In scope:

- Move the current CAM MCP server records into `~/repos/awesome-mcp-servers`.
- Preserve the installable schema fields CAM needs for registry-based install.
- Add MCP catalog source configuration to CAM.
- Replace the production embedded MCP registry with a configured source-backed loader.
- Keep CLI and desktop MCP flows working with minimal public API disruption.
- Update user-facing wording from "bundled registry" to "catalog registry" where relevant.

Out of scope:

- Changing how CAM writes MCP server entries into client config files.
- Reworking the desktop MCP page UX beyond source-backed data loading.
- Changing manual custom MCP add/remove/list behavior.
- Redesigning non-MCP catalog systems.

## Architecture

Ownership boundaries:

- `awesome-mcp-servers` owns MCP server data, including the existing 381 CAM server records and their install/run metadata.
- CAM owns MCP client integration: listing installed servers, adding/removing servers from client configs, choosing preferred installations, and exposing CLI/desktop APIs.
- CAM config owns source selection through a local-first, remote-second catalog source list.

CAM's MCP registry package should retain the existing registry abstraction where possible:

- `Registry.Get`
- `Registry.All`
- `Registry.Search`
- `Registry.Names`
- `ServerSchema.PreferredInstallation`

The current production dependency on `internal/mcp/registry/servers` should be removed. Tests may use small fixtures, but CAM should not keep a production embedded copy of all MCP server records.

Key design decisions:

- MCP catalog sources should use the same repository-source mental model as instructions, skills, agents, and plugins, rather than introducing a separate config system.
- The registry API should stay stable for callers. Source loading is an implementation detail below the registry package.
- CAM should support multiple source artifact shapes during the transition so `awesome-mcp-servers` can evolve its public dist format without forcing another CAM API change.
- Local catalog sources must remain useful for development, private entries, and emergency overrides.
- CAM should not auto-write migrated catalog data back into user config. User config only selects sources.

## Components

### `awesome-mcp-servers`

Move or transform CAM's current MCP JSON records into `~/repos/awesome-mcp-servers` so that repo can validate, build, and publish the MCP catalog.

The published dist artifact must preserve CAM's installable schema requirements. CAM should accept either:

- a wrapped dist object with metadata plus a server collection, or
- a direct array/map of server schema objects.

Each installable server record must preserve these fields when available:

- `name`
- `display_name`
- `description`
- `repository`
- `homepage`
- `author`
- `license`
- `categories`
- `tags`
- `installations`
- optional `arguments`

Required fields for CAM installability are:

- `name`
- `description`
- at least one usable installation entry

`display_name` should default to `name` when omitted. Optional metadata should be preserved for search, display, and future desktop filtering, but missing optional metadata must not block installation.

Recommended dist wrapper shape:

```json
{
  "schema_version": 1,
  "generated_at": "2026-06-22T00:00:00Z",
  "source": "Chat2AnyLLM/awesome-mcp-servers",
  "servers": [
    {
      "name": "example",
      "display_name": "Example",
      "description": "Example MCP server",
      "installations": []
    }
  ]
}
```

CAM should ignore wrapper metadata it does not understand, but it should reject a wrapper that lacks a recognizable server collection.

### CAM config

Add an MCP catalog source section to CAM's bundled/user config model. The default should mirror other catalogs: local override first, remote source second.

Example shape:

```yaml
repositories:
  mcpServers:
    sources:
      - type: local
        path: ~/.config/code-agent-manager/mcp_servers.json
      - type: remote
        url: https://raw.githubusercontent.com/Chat2AnyLLM/awesome-mcp-servers/main/dist/servers.json
```

Use `code_agent_manager` paths if package/config files need to be updated. Do not introduce new references to the old `code_assistant_manager` package path.

Config semantics:

- `repositories.mcpServers.sources` is optional for backwards compatibility.
- If the user config omits MCP sources, CAM falls back to the bundled default config.
- If both user and bundled config define MCP sources, the user config wins in the same way existing repository-source settings win.
- Relative local paths should resolve consistently with existing repository-source behavior.
- The remote URL should point at a versioned or branch-based dist artifact from `Chat2AnyLLM/awesome-mcp-servers`; the first implementation can use `main` if that matches other CAM defaults.

Do not add per-command flags for catalog source selection in this migration. Source selection belongs in CAM config so CLI and desktop behavior stays consistent.

### CAM MCP loader

Replace `LoadBundledRegistry` usage with a source-backed loader, for example `LoadRegistry`.

The loader should:

- read MCP catalog sources from `camconfig`
- load local JSON sources when present
- fetch remote JSON sources with the existing cache behavior
- merge records deterministically
- normalize records into the current `ServerSchema`
- return clear errors for malformed sources

Local sources have priority. Remote sources fill missing entries but do not override local entries.

Loader responsibilities in more detail:

1. Resolve MCP source config from `camconfig`.
2. For each source, read bytes from local disk or remote cache/network.
3. Decode JSON into one of the supported source artifact shapes.
4. Normalize every decoded record into `ServerSchema`.
5. Validate required installable fields.
6. Merge by canonical server name, preserving the first valid record.
7. Return a `Registry` with deterministic ordering from the merged map.

Canonical names should use the existing registry naming rules. If the current registry treats names case-sensitively, keep that behavior unless tests demonstrate otherwise. Search behavior should continue matching the existing `Registry.Search` semantics.

Supported artifact shapes:

- direct array: `[{...server...}]`
- direct map: `{ "server-name": {...server...} }`
- wrapped object with one of these collection keys: `servers`, `items`, or `data`

When loading a direct map, the map key may supply `name` only when the record omits `name`. If both are present and conflict, fail that record/source instead of guessing.

Remote caching should reuse existing cache directory, TTL, and fetch behavior used by other configured repository sources where possible. Avoid introducing a separate HTTP/cache implementation unless the existing abstraction cannot handle plain JSON artifacts.

### CLI and desktop

CLI and desktop code should continue using the registry abstraction rather than knowing source details.

Update flows that currently call `mcp.LoadBundledRegistry()`:

- `cam mcp search`
- `cam mcp add NAME --client ...` when resolving catalog servers
- `cam mcp server list/search/show`
- desktop `SearchRegistry`
- desktop `ShowServer`
- desktop `ListRegistry`
- desktop `InstallFromRegistry`

Manual MCP operations that do not require the catalog must keep working offline:

- `cam mcp list`
- `cam mcp add --command ...`
- `cam mcp add --url ...`
- `cam mcp remove`

User-facing wording should consistently distinguish:

- **catalog servers**: installable definitions loaded from configured catalog sources
- **installed servers**: entries already present in a client config
- **custom servers**: manually added command or URL entries

Avoid saying the registry is "bundled" in CLI help, errors, desktop labels, or tests once the production embedded registry is removed.

## Compatibility and migration behavior

This migration should be source-compatible for existing CLI and desktop callers where practical:

- Existing commands and method names remain available.
- Existing output DTO shapes remain available.
- Existing installed MCP server configs are not rewritten.
- Existing manual MCP add/remove/list behavior is unchanged.
- Existing registry install behavior should produce the same client config entry for the same server definition.

Expected user-visible change:

- Registry/list/search/install commands now depend on configured catalog sources instead of the embedded production registry.
- Error messages reference catalog loading failures instead of bundled registry failures.

If users rely on offline registry access, they should configure a local catalog JSON source or rely on a fresh remote cache. This design does not require CAM to ship a full production fallback copy inside the binary.

## Data flow

### Listing and searching catalog servers

1. User runs a registry command or opens the desktop registry page.
2. CAM calls the source-backed MCP registry loader.
3. The loader walks configured MCP catalog sources in order.
4. Local entries are loaded first.
5. Remote entries are loaded from cache or network and fill missing names.
6. Records are normalized into `ServerSchema`.
7. Existing list/search APIs return sorted registry results.

### Installing a catalog server

1. User selects a server by name from CLI or desktop.
2. CAM loads the configured MCP registry.
3. CAM finds the `ServerSchema` by name.
4. Existing `ServerFromSchema` converts the preferred installation into a client config server entry.
5. Existing `AddServer` writes the entry into the target client and scope.

### Migrating the data

1. Copy or transform current CAM MCP records from `internal/mcp/registry/servers` into `~/repos/awesome-mcp-servers`.
2. Update `awesome-mcp-servers` schema/build logic if needed so `dist/servers.json` includes installable schema fields.
3. Wire CAM to load the dist artifact from configured sources.
4. Remove CAM's production embedded MCP server record directory.

Data migration checklist:

- Preserve all 381 existing CAM records unless an individual record is known to be invalid.
- Preserve server names exactly to avoid breaking scripts that install by name.
- Preserve installation order inside each server because `PreferredInstallation` may depend on it.
- Preserve package manager command/args/env fields without shell re-interpretation.
- Keep provenance in `awesome-mcp-servers` commit history rather than adding CAM-only migration metadata to each record.
- Compare generated `dist/servers.json` against CAM's current registry count before removing embedded production data.

## Error handling

- Missing local source: skip it.
- Malformed local source: fail catalog loading with a clear parse/source error.
- Remote unavailable with fresh cache: use the cache.
- Remote unavailable with no usable cache: fail registry list/search/install-from-registry with a clear catalog loading error.
- Duplicate server names: earlier/local sources win; later/remote sources fill only missing names.
- Unsupported record shape: fail the source with an error naming the missing required fields or unsupported format.
- Installed-server operations: continue to work without the catalog when they only read/write local MCP client configs.

Avoid silently dropping malformed installable records. A broken catalog should be visible during registry operations so users do not install incomplete MCP definitions.

Error messages should include:

- source type and path/URL
- JSON shape problem, parse problem, or validation problem
- server name when the error is record-specific
- whether CAM attempted cache fallback for remote sources

Do not include secrets from environment variables, request headers, or private local paths beyond the configured catalog path itself.

## Implementation plan

1. Add MCP source config fields and bundled defaults under the existing CAM config model.
2. Add a source-backed MCP registry loader behind the existing registry abstraction.
3. Update CLI and desktop registry call sites from `LoadBundledRegistry` to the new loader.
4. Import/migrate CAM MCP records into `awesome-mcp-servers` and ensure its dist build emits CAM-installable fields.
5. Point CAM's bundled default MCP catalog source at the published `awesome-mcp-servers` dist artifact.
6. Replace production embedded registry data with small test fixtures only.
7. Update tests, CLI help, desktop labels, docs, and README wording from bundled registry to catalog registry.
8. Run required CAM and `awesome-mcp-servers` verification.

Implementation should keep each step separately reviewable. Do not combine data migration, loader changes, and wording cleanup into one hard-to-review patch if avoidable.

## Acceptance criteria

- CAM production code no longer imports or embeds `internal/mcp/registry/servers` as the full MCP catalog.
- CAM default config includes `Chat2AnyLLM/awesome-mcp-servers` as the remote MCP catalog source.
- CAM can load MCP catalog records from local JSON, remote JSON, and cached remote JSON.
- CAM accepts direct array, direct map, and wrapped dist catalog shapes.
- Local source entries override remote entries by server name.
- CLI registry list/search/show/add behavior remains functionally equivalent for migrated records.
- Desktop registry search/list/show/install behavior keeps the same DTO contracts.
- Manual installed/custom MCP operations work without catalog access.
- User-facing registry wording no longer claims the full registry is bundled.
- `awesome-mcp-servers` contains the migrated CAM records and can build/validate the published dist artifact.

## Risks and mitigations

- **Remote catalog unavailable**: reuse existing cache behavior and support local override sources.
- **Schema drift between repos**: accept multiple artifact wrapper shapes and validate only CAM-required installable fields.
- **Record loss during migration**: compare server counts and names before removing CAM's embedded production records.
- **Different install output after migration**: preserve installation arrays and verify representative installs through CLI tests.
- **User confusion about offline behavior**: update help/errors/docs to explain configured catalog sources and cache fallback.
- **Duplicated source-loading code**: prefer existing repository-source/cache abstractions so MCP follows the same pattern as other CAM catalogs.

## Rollback plan

If the source-backed loader causes production breakage before release, rollback should restore the previous `LoadBundledRegistry` call path and embedded production data while keeping the migrated `awesome-mcp-servers` data intact. After release, rollback should instead be done by updating the configured catalog source to a known-good local or remote artifact, because CAM should no longer own the production catalog copy.

## Testing

### CAM loader tests

- Load a direct array of `ServerSchema` records.
- Load a direct map of server name to `ServerSchema`.
- Load a wrapped dist object from `awesome-mcp-servers`.
- Merge local and remote sources with local priority.
- Reject malformed installable schemas with clear errors.
- Use cached remote data when fresh enough.

### CLI tests

Update existing MCP registry tests from bundled wording to catalog wording.

Ensure:

- `cam mcp server list` lists catalog servers.
- `cam mcp server search QUERY` finds catalog matches.
- `cam mcp server show NAME` prints schema JSON.
- `cam mcp add NAME --client claude` resolves a catalog server and installs it.
- Manual custom MCP add/remove/list tests remain unchanged.

### Desktop tests

Ensure these methods continue returning the same DTO shape through the new loader:

- `SearchRegistry`
- `ListRegistry`
- `ShowServer`
- `InstallFromRegistry`

### Data migration tests

- CAM no longer depends on production files under `internal/mcp/registry/servers`.
- CAM config includes the `Chat2AnyLLM/awesome-mcp-servers` remote dist URL.
- `awesome-mcp-servers` build/validation succeeds after importing CAM MCP records.

## Verification

Because this change affects Go code and catalog data, implementation should run relevant tests one by one as required by project instructions. After code changes, reinstall CAM with:

```sh
rm -rf dist/*
./install.sh uninstall
./install.sh
```

No tests are required for this design-only spec change.
