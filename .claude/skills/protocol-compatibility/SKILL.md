---
name: protocol-compatibility
description: Add Cedrazeka wire and SDK regression scenarios for transports, server API, proxies, or subscriptions.
---

# Protocol compatibility

Read `docs/testing/compatibility-suite.md` and locate the corresponding feature ID. Pick the existing transport entry point and add evidence through `test/compat/scenarios`. Assert codes, field presence, offsets, frame order, and actual delivery before comparing transcripts. Keep normalization limited to documented volatile values. Run diff and golden for each affected profile. Record only an individually investigated scenario, with the reason in the proposed commit message.
