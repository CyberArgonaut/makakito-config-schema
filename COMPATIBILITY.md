# Compatibility

This file tracks which schema versions are compatible with which consumer repos and their minimum required versions.

When a new schema version is released:
1. Update this table with the new row.
2. Notify consumer repo maintainers.
3. Consumers pin their `go.mod` (or `package.json`) to the appropriate major version.

## Breaking changes policy

- **Patch** (`1.0.x`): backwards-compatible fixes (documentation, typos in descriptions). No consumer update needed.
- **Minor** (`1.x.0`): additive changes (new optional fields, new enum values). Consumers must update to accept new fields; existing configs remain valid.
- **Major** (`x.0.0`): breaking changes (renamed or removed required fields, tightened enum values). Consumers must update before using new configs.

## Compatibility table

| Schema version | `makakito-cli` | `makakito-runner` | `makakito-builder` (TS) | Notes |
|---|---|---|---|---|
| **1.0.0** | TBD | TBD | TBD | Initial release. Consumer repo versions TBD until those repos are scaffolded. |

## How consumers import this repo

**Go** (`makakito-cli`, `makakito-runner`):

```bash
go get github.com/CyberArgonaut/makakito-config-schema/generated/go/config@v1
```

Pin to the major version (`v1`) to receive non-breaking updates automatically.

**TypeScript** (`makakito-builder`):

Generate types locally from the published schema using `json-schema-to-typescript` or an equivalent tool. Pin the schema version using the `$id` field in `schema/config.schema.json`.
