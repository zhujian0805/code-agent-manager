# awesome_prompts Source-of-Truth Design

Date: 2026-06-22

## Goal

Refactor CAM's Prompts feature so prompts come from `Chat2AnyLLM/awesome-prompts` as the single source of truth. The authoritative prompt data is the repository's generated JSON file:

```text
https://raw.githubusercontent.com/Chat2AnyLLM/awesome-prompts/master/dist/prompts.json
```

CAM should also bundle a copy of that JSON at build time so the prompts sync path still works offline.

## Current State

`internal/prompts/service.go` currently syncs prompts from three sources:

1. `claude` — embedded `internal/prompts/embed/claude_library.json`, mirrored from Claude's prompt library.
2. `prompts_chat` — fetched from `https://raw.githubusercontent.com/f/prompts.chat/main/prompts.csv`.
3. `promptingguide` — fetched from markdown files in `dair-ai/Prompt-Engineering-Guide` and parsed heuristically.

This means prompt quality and structure depend on multiple upstream formats. The new design replaces these sources with the curated `awesome-prompts` JSON feed.

## Source Model

Use a single prompt source identifier:

```text
awesome_prompts
```

Display name:

```text
Awesome Prompts
```

Remote source:

```text
https://raw.githubusercontent.com/Chat2AnyLLM/awesome-prompts/master/dist/prompts.json
```

Bundled fallback:

```text
internal/prompts/embed/awesome_prompts.json
```

The bundled file mirrors `dist/prompts.json` and is embedded into the Go binary with `//go:embed`.

## Data Shape

`awesome-prompts` currently exposes this JSON shape:

```json
{
  "version": "1.0.0",
  "generated_at": "2026-06-22T05:40:04Z",
  "count": 3,
  "prompts": [
    {
      "slug": "code-reviewer",
      "title": "Code Reviewer",
      "description": "Reviews code for bugs, security issues, and improvements with structured feedback.",
      "prompt": "...",
      "tags": ["developer", "code-review", "quality"],
      "category": "developer-tools",
      "author": "zhujian0805",
      "variables": []
    }
  ]
}
```

CAM will map each prompt into the existing `prompts.Prompt` model.

| awesome-prompts field | CAM field | Mapping |
| --- | --- | --- |
| `slug` | `source_url` | `https://github.com/Chat2AnyLLM/awesome-prompts/blob/master/prompts/<slug>.yaml` |
| `title` | `title` | Direct |
| `description` | `description` | Direct |
| `prompt` | `content` | Direct |
| `tags[]` | `tags` | Join with `, ` |
| `category` | `category` | Direct |
| `author` | `author` | Direct |
| `variables[]` | ignored | Existing CAM schema has no variables field |
| constant | `source` | `awesome_prompts` |

`slug` is unique in `awesome-prompts`, so using it in `source_url` keeps CAM's existing unique index on `(source, source_url)` valid.

## Backend Design

### Service

Refactor `internal/prompts/service.go` around one source.

Remove source-specific fetchers and parsers for:

- `claude`
- `prompts_chat`
- `promptingguide`

Add an `awesomePromptsLibrary` parser for the JSON shape above.

`SyncAll(ctx)` should:

1. Fetch the remote JSON from `dist/prompts.json`.
2. If the remote fetch succeeds and parses, sync the remote prompts.
3. If the remote fetch fails or returns invalid JSON, fall back to the embedded `awesome_prompts.json`.
4. Upsert every mapped prompt with `source = "awesome_prompts"`.
5. Delete stale rows from retired sources: `claude`, `prompts_chat`, and `promptingguide`.
6. Return the number of prompts successfully upserted.

`RefreshSource(ctx, source)` should accept only `awesome_prompts`. Unknown sources should continue to return an error.

`GetSourceStatus(ctx)` should return one source status for `awesome_prompts`.

### Store

Add a small store helper to delete prompts by source, or delete retired prompt sources in one statement. This keeps source cleanup explicit and testable.

The existing prompt table schema remains unchanged.

### Sidecar API

Keep the existing API surface:

- `GET /api/prompts`
- `POST /api/prompts`
- `GET /api/prompts/search?q=...`
- `POST /api/prompts/sync`
- `GET /api/prompts/sources`

`POST /api/prompts/sync` should work both with an empty source and with `{ "source": "awesome_prompts" }`.

`GET /api/prompts/sources` should return one enabled source:

```json
[
  {
    "source": "awesome_prompts",
    "name": "Awesome Prompts",
    "last_sync": "",
    "prompt_count": 3,
    "enabled": true
  }
]
```

## Frontend Design

Keep the existing Prompts page layout:

- search input
- source filter dropdown
- Sync button
- paginated expandable table
- View source link
- Copy prompt action

With one source, the dropdown will contain only `Awesome Prompts`, plus the existing `All sources` option. Keeping the dropdown avoids unnecessary UI churn and preserves room for future JSON-backed sources if needed.

Update the source label mapping in `frontend/src/pages/Prompts.tsx`:

```ts
case 'awesome_prompts': return 'Awesome Prompts'
```

No i18n key changes are required because existing prompt UI strings are source-agnostic.

## Error Handling

Remote fetch failures should not make sync fail if the embedded fallback parses successfully. In that case, CAM syncs the bundled prompts and returns success.

Sync should return an error only when both remote JSON and embedded JSON fail to parse or map.

Malformed prompt entries should be skipped if they lack required fields such as `slug`, `title`, or `prompt`. Sync should continue with valid entries.

## Tests

Update `internal/prompts/service_test.go`:

- Replace Claude-specific embedded tests with awesome-prompts embedded tests.
- Verify the bundled JSON parses.
- Verify bundled prompt entries have non-empty `slug`, `title`, and `prompt`.
- Verify sync stores every bundled prompt.
- Verify remote JSON maps fields correctly.
- Verify remote fetch failure falls back to embedded JSON.
- Verify retired sources are removed on sync.

Update `internal/sidecar/api_coverage_test.go`:

- Replace the sync coverage case using `{ "source": "claude" }` with `{ "source": "awesome_prompts" }`.

## Out of Scope

- Supporting `variables[]` in the CAM prompt table.
- Importing bulk scraped prompt outputs from `awesome-prompts/scraped`.
- Replacing skills, plugins, or subagents in this implementation step.
- Automatically syncing prompts on sidecar startup.

## Files Expected to Change

- `internal/prompts/service.go`
- `internal/prompts/store.go`
- `internal/prompts/service_test.go`
- `internal/prompts/embed/awesome_prompts.json`
- `internal/prompts/embed/claude_library.json` — remove
- `internal/sidecar/prompts_handler.go`
- `internal/sidecar/api_coverage_test.go`
- `frontend/src/pages/Prompts.tsx`
