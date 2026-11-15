# ADR-0024: Undocumented PRO behavior is out of mandatory scope — DEC-06

- Status: **accepted** (user decision DEC-06, 2026-11-15)
- Date: 2026-11-15
- Feature IDs: PRO-29, PRO-30, PRO-27.05; every row with a `PRO-OQ-*` reference

## Decision
- The mandatory scope is PRO features publicly documented at docs@69da714 or in the PRO release notes up to v6.10.3, and **listed** on the site.
- Excluded:
  - PRO-29 ephemeral client publications (unlisted page);
  - PRO-30 dictionary compression (`draft: true`);
  - PRO-27.05 shared poll relay (WIP).
  
  They are tracked as `excluded (not released)`. The decoder keeps tolerating the draft protocol fields (`flag`, `dict`, `profile`), which the fork already does.
- For a documented PRO feature with undocumented details, the undocumented part becomes an open question (`PRO-OQ-*`). It is never guessed. If we still need behavior there, we choose it and label it **EXTENSION**, never "compatible".
- Re-check at each re-baseline (baseline.md §Re-baselining). A feature that becomes documented moves into scope through an ADR.

## Sources
User decision DEC-06; pro-feature-inventory.md §gaps; PRO-OQ-01, PRO-OQ-23.
