# ADR-0017: Configuration system

- Status: revised 2026-11-15 (DEC-01, DEC-04)
- Date: 2026-11-15

## Revision 2026-11-15 (user decisions)
The configuration system is inherited. Upstream uses viper v1.21.0 with go-viper/mapstructure, yaml.v3, go-toml/v2 and an in-tree `envconfig` fork (MIT, Kelsey Hightower), so the 'no viper' decision is void. The env prefix comes from `internal/brand`: `CEDRAZEKA_` by default, `CENTRIFUGO_` in compatibility mode (ADR-0022). PRO keys are added to upstream `configtypes`.

The original Phase 0 text below is kept for history.
- Feature IDs: OSS-CFG-*, OSS-CLI-*

## Decision
- A single typed `Config` struct with explicit field tags for the JSON/YAML/TOML key names and defaults. One source of truth generates `defaultconfig`, `defaultenv` and `configdoc`, and gives the redaction flags for the config viewer (PRO-08).
- Loading: file (format by extension) < env (`CENTRIFUGO_*` naming for drop-in compatibility; to be confirmed with the user given the branding rule) < flags.
  - Env decoding follows the documented rules: ParseBool, space-separated `[]string`, JSON for objects and maps, `CENTRIFUGO_VAR_` substitution.
  - Unknown keys produce WARN, and errors under `checkconfig --strict`.
- Parsers: `gopkg.in/yaml.v3`, `github.com/BurntSushi/toml` (or pelletier/go-toml/v2), and stdlib `encoding/json`. Viper is not used: its key-case folding and merge semantics would be hard to match precisely and test.
- Reload: SIGHUP reloads only token secrets and channel options (documented). Changes that need restarting a subsystem are rejected, as v6.10.3 does.

## Open
The env prefix `CENTRIFUGO_` is needed for drop-in compatibility but uses the upstream brand name. User decision Q4 in open-questions.

## Sources
docs/server/configuration.md, console_commands.md.
