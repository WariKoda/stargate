# Security Policy

Stargate reads your SSH configuration and launches the native OpenSSH client.
That places it directly on the path to your servers, so the security model is
documented here in full.

## Supported versions

Stargate is pre-1.0. Security fixes are applied to the latest released version
only. Please reproduce a report against the newest tag or `main` before filing.

## Reporting a vulnerability

Please do **not** open a public issue for a security problem.

Report it privately through GitHub's
[private vulnerability reporting](https://github.com/warikoda/stargate/security/advisories/new)
(Security → Report a vulnerability). If that form is unavailable to you, open a
regular issue asking for a private contact channel — **without** any details of
the vulnerability itself.

Please include the affected version, your operating system and OpenSSH version,
a minimal `ssh_config` snippet that reproduces the issue (with real hostnames
replaced), and the impact you believe it has.

This is a spare-time project, not a funded one. Expect an acknowledgement within
about a week. There is no bug bounty. If a fix is warranted, it will be released
with a GitHub Security Advisory crediting you, unless you prefer otherwise.

## Security model

### What Stargate does not touch

- It never reads, prompts for, stores, or logs **passwords**.
- It never reads **private key material**. `IdentityFile` is treated as an
  opaque path for display only; the file contents are never opened.
- It never talks to the **SSH agent** and never handles agent-forwarded
  credentials.
- It performs **no network authentication** of its own.

All authentication is handled entirely by the native `ssh` client, which
inherits the terminal directly.

### How connections are launched

With the default `~/.ssh/config`, Stargate invokes:

```text
ssh -- <selected-alias>
```

With `--config` or `STARGATE_CONFIG`, it invokes:

```text
ssh -F <resolved-config-path> -- <selected-alias>
```

- The command is executed via `os/exec` with an explicit argument vector.
  **No shell is involved**, so no shell metacharacter in a config value can be
  interpreted as a command.
- Without an explicit config, `ssh` loads `~/.ssh/config` and the system-wide
  `/etc/ssh/ssh_config` exactly as it would when run directly. System policies
  such as crypto policies, `StrictHostKeyChecking`, or global known-hosts files
  therefore stay in effect.
- An explicit config is passed with `-F`, which pins the client to the file
  Stargate parsed. OpenSSH then ignores `/etc/ssh/ssh_config`; this is
  OpenSSH's documented behavior for `-F`.
- The `--` separator prevents an alias beginning with `-` from being parsed as
  an ssh option.
- Aliases are validated before use: empty aliases, aliases containing NUL bytes,
  and aliases containing whitespace are rejected.
- Stargate connects by **alias**, never by a reconstructed `user@host:port`.
  This is deliberate: it lets the native client apply the full effective
  configuration — `ProxyJump`, `ProxyCommand`, forwardings, identities and host
  key policy — as OpenSSH would.

### `Match exec` is never executed

An SSH config can ask for an arbitrary shell command via `Match exec`. Stargate
**never executes it**. When parsing, Stargate skips `Match exec` blocks and all
other `Match` blocks it cannot evaluate, together with any `Include` inside
them. This is a security decision, not only a scope decision: parsing a config
file must not be able to run code. OpenSSH still evaluates these blocks when
it connects, as it would without Stargate.

### Command preview and clipboard

`--print` and `Ctrl+P` render the invocation as a shell-safe string, and
`Ctrl+Y` copies it to the system clipboard. This string is produced for display
and copying only — Stargate never executes it, and never round-trips it back
through a shell. Note that copying places the target hostname and username on
your system clipboard, which other local applications can read.

### Connectivity checks

`--check` and `Ctrl+R` open a plain TCP connection to the effective `HostName`
and `Port`, with a two-second timeout and a bounded number of concurrent
connections. These checks:

- do **not** authenticate and send no credentials,
- do **not** complete an SSH handshake,
- do **not** traverse `ProxyJump` or `ProxyCommand` routes.

A failed check therefore does not mean a host is unreachable — a host reachable
only through a jump host will correctly show as failing a *direct* check. Be
aware that checks generate outbound TCP connections to every displayed host,
which may be visible to network monitoring.

### Trust boundary

Stargate trusts your SSH config file as much as `ssh` itself does: it is your
own configuration. A hostile `~/.ssh/config` is outside the threat model with
one explicit exception — parsing it must never execute code, which is why
`Match exec` is never evaluated.

Stargate has no network listeners, no telemetry, no auto-update, and no
configuration of its own beyond a config-file path.
