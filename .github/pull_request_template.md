## What does this change?

<!-- What changed and why. Link any related issue: "Fixes #123". -->

## How was it tested?

<!--
Unit tests are not enough for TUI changes. Note what you exercised manually,
e.g. filtering, connecting, and returning cleanly from a session.
-->

## Checklist

- [ ] `make check` passes and the code is `gofmt`-formatted
- [ ] Tests cover the changed behavior
- [ ] No real hostnames, usernames, or IP addresses in code, tests, or docs
- [ ] Documentation updated if behavior, keybindings, or flags changed
- [ ] If `internal/ssh` changed: [SECURITY.md](https://github.com/nibra180/stargate-tui/blob/main/SECURITY.md) still
      describes the command construction accurately
