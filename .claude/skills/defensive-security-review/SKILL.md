---
name: defensive-security-review
description: Review Cedrazeka authentication, authorization, configuration, admin actions and data handling in code.
---

# Defensive security review

Read the applicable section of `docs/security/threat-model.md` and the relevant ADR. Trace the server-side boundary through the handler to policy and storage. Check deny-by-default behavior, tenant scoping, bounded work, audit coverage and redaction. For JWT/JWKS/OIDC or CEL changes verify the intended validations with local unit tests. Report findings with file references and remediation.
