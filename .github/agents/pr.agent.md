---
# Fill in the fields below to create a basic custom agent for your repository.
# The Copilot CLI can be used for local testing: https://gh.io/customagents/cli
# To make this agent available, merge this file into the default repository branch.
# For format details, see: https://gh.io/customagents/config

name:
description:
---

# My Agent
Consider the following for each PR.
### 1. PR Type / Mode

- [ ] Enhancement (new feature or improvement)
- [ ] Debug / Bugfix
- [ ] Refactor / Cleanup
- [ ] Performance
- [ ] Documentation / Comments
- [ ] CI / Infra / Config
- [ ] Other: __________________

### 2. Summary

> High-level: what does this PR do and why?

- Problem / motivation:
- High-level solution:
- Scope: (what’s in / out of this PR)

### 3. Technical Details

- Key changes (modules, functions, APIs touched):
- Data structures / schemas updated:
- Any backwards-incompatible changes?

### 4. Behavior & UX (if applicable)

- New behavior:
- Existing behavior changed?
- CLI / API / UI examples (before → after):

### 5. Tests & Verification

- [ ] Unit tests added
- [ ] Integration / E2E tests
- [ ] Documentation

Describe what you actually ran:

```text
Commands:
- ...

Scenarios:
- ...
