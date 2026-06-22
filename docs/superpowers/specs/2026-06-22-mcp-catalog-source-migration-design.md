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

## Error handling

- Missing local source: skip it.
- Malformed local source: fail catalog loading with a clear parse/source error.
- Remote unavailable with fresh cache: use the cache.
- Remote unavailable with no usable cache: fail registry list/search/install-from-registry with a clear catalog loading error.
- Duplicate server names: earlier/local sources win; later/remote sources fill only missing names.
- Unsupported record shape: fail the source with an error naming the missing required fields or unsupported format.
- Installed-server operations: continue to work without the catalog when they only read/write local MCP client configs.

Avoid silently dropping malformed installable records. A broken catalog should be visible during registry operations so users do not install incomplete MCP definitions.

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
