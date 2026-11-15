# Naming, trademarks and compatibility mode — legal notes

Status: engineering notes, **not legal advice**. A legal review is required before public release (risk R-LEGAL in the overview).

## Facts
- "Centrifugo" and "Centrifugal" are names used by Centrifugal Labs LTD for its products. We did not check trademark registrations; that check is part of the legal review.
- The Apache-2.0 license §6 grants **no** trademark rights beyond "reasonable and customary use in describing the origin of the Work and reproducing the content of the NOTICE file".
- The MIT and BSD licenses of PRV-02..04 and PRV-10..11 grant no trademark rights either.

## What we do (ADR-0022)
| Surface | Default mode | Compatibility mode | Rationale |
|---|---|---|---|
| Product, binary, Docker image, UI, docs title | Cedrazeka / `cedrazeka` (DEC-12) | own name | never use the upstream brand as our name |
| Env prefix | `CEDRAZEKA_` | `CENTRIFUGO_` | operator compatibility; an identifier, not marketing |
| Metric namespace | `cedrazeka` | `centrifugo` | dashboard and alert compatibility |
| Default key prefixes | `cedrazeka` | `centrifugo` | operator compatibility; no mixed clusters (ADR-0023) |
| gRPC service full names | own package | upstream names **also** registered | wire compatibility with existing generated clients and proxy backends |
| Kafka header names | own | upstream | wire compatibility |
| License/copyright notices | unchanged | unchanged | required by the licenses |

## Allowed wording (pending legal review)
- "Includes software from Centrifugo OSS by Centrifugal Labs LTD (Apache-2.0)" in NOTICE, README and about pages. This is origin attribution.
- "Wire-compatible with the Centrifugo v6 client protocol and server API" in technical docs. This is nominative and factual, with no logo and no implied endorsement.

## Not allowed
- Calling the product, or any edition of it, "Centrifugo", "Centrifugo PRO" or a confusingly similar name.
- Upstream logos, colors or UI text (CLAUDE.md).
- Implying endorsement by, or affiliation with, Centrifugal Labs.
- Describing a feature as "Centrifugo PRO compatible" when it is not documented upstream (ADR-0024, ADR-0025).

## Open
- DEC-12 decided 2026-11-15: product name **Cedrazeka**; identifiers `cedrazeka` / `CEDRAZEKA_`. We did not run a trademark search on it; that belongs to the pre-release legal review.
- Whether compatibility mode itself (exposing `CENTRIFUGO_*` names) needs explicit legal sign-off. Recommended: yes, before the first public release.
