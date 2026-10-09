# Stargate

Stargate is a terminal user interface for browsing, filtering, and opening SSH
connections defined in `~/.ssh/config`.

It delegates connections to the native OpenSSH client. Existing SSH features
such as `ProxyJump`, `ProxyCommand`, agent forwarding, identities, and port
forwarding therefore continue to work as configured.

## Requirements

- Unix-like operating system (Linux or macOS)
- OpenSSH client (`ssh`) available in `$PATH`
- Interactive terminal on stdin and stdout
- Go 1.24.2 or newer when installing from source

## Installation

Install the latest release:

```sh
go install github.com/warikoda/stargate/cmd/gate@latest
```

Ensure that your Go binary directory (`$GOBIN`, or usually `$(go env
GOPATH)/bin`) is included in `$PATH`.

Install the current local checkout:

```sh
make install
```

## Usage

Use the default SSH config:

```sh
gate
```

Use a different config file:

```sh
gate --config /path/to/ssh_config
```

Print the exact, shell-safe OpenSSH command without starting the TUI:

```sh
gate --print production
```

Start asynchronous direct TCP checks for the displayed host endpoints:

```sh
gate --check
```

The config path is selected in this order:

1. `--config PATH`
2. `STARGATE_CONFIG`
3. `~/.ssh/config`

### Controls

| Action | Keyboard | Mouse |
|---|---|---|
| Navigate | `↑`/`↓` | Scroll wheel |
| Filter | `/`, then type | — |
| Apply filter | `Enter` | — |
| Clear/cancel filter | `Esc` | — |
| Connect | `Enter` | Left-click a host |
| Preview SSH command | `Ctrl+P` | — |
| Copy SSH command | `Ctrl+Y` | — |
| Check selected host | `Ctrl+R` | — |
| Toggle extended help | `?` | — |
| Quit | `q` or `Ctrl+C` | — |

Filtering starts only after `/` is pressed. While browsing, `q` quits and `Esc`
has no effect. While filtering, `Esc` cancels or clears the filter.

When the terminal has enough room, a detail panel beside or below the list
shows the selected host's effective hostname, user, port, identity file, source
config file, and direct, `ProxyJump`, or `ProxyCommand` route. Long values are
truncated to the available width, and the panel is hidden in short or very
narrow terminals so the host list remains usable.

When an SSH session ends, Stargate restores the full, unfiltered host list,
keeps the connected host selected, and remains available for the next connection.
Command copying uses the operating system clipboard and reports an error when no
supported clipboard utility is available.

Press `Ctrl+R` to check the selected host, or start `gate --check` to check all
hosts automatically. Connectivity checks run concurrently with a two-second
timeout and a bounded number of connections. They only test direct TCP access to the
effective `HostName` and `Port`; they do not authenticate and do not traverse
`ProxyJump` or `ProxyCommand` routes. A failed direct check therefore does not
necessarily mean that a host configured through a jump host is unreachable.

### Theme colors

Stargate uses the terminal palette by default and detects Omarchy themes from
`$XDG_STATE_HOME/omarchy/current/theme/colors.toml` (normally
`~/.local/state/omarchy/current/theme/colors.toml`). The older location below
`$XDG_CONFIG_HOME` is supported as a fallback.

Set `STARGATE_THEME` to `auto`, `ansi`, `omarchy`, or `default`. `auto` is the
default: it loads the current Omarchy colors when available and otherwise uses
ANSI terminal colors. `default` restores Stargate's original violet palette.
`STARGATE_THEME_FILE` can point to another Omarchy-compatible `colors.toml`.

## SSH config behavior

- `Host` blocks with concrete aliases are listed.
- Multiple concrete aliases on one `Host` line are listed separately.
- Wildcard and negated patterns (`*`, `?`, `!`) are used for effective settings
  but are not shown as connection targets.
- `Include` directives and glob patterns are followed recursively.
- Effective `HostName`, `User`, `Port`, `IdentityFile`, `ProxyJump`, and
  `ProxyCommand` settings use OpenSSH's first-value-wins behavior.
- Hosts using `ProxyJump` or `ProxyCommand` are marked `via Jump-Host`.
- `HostName` values support the `%h` and `%%` tokens.
- `Match all` and `Match host <patterns>` blocks contribute effective settings.
  Stargate skips every other `Match` block, including `Match exec`, without
  evaluating it. Their settings are not shown in the TUI, but OpenSSH still
  applies them when connecting.

With the default `~/.ssh/config`, Stargate invokes connections as:

```text
ssh -- <selected-alias>
```

OpenSSH then reads `~/.ssh/config` and the system-wide `/etc/ssh/ssh_config`
as usual. With `--config` or `STARGATE_CONFIG`, Stargate uses:

```text
ssh -F <resolved-config-path> -- <selected-alias>
```

`-F` makes OpenSSH ignore `/etc/ssh/ssh_config`, so system-wide settings such
as crypto policies or `StrictHostKeyChecking` do not apply to these
connections.

Stargate does not read passwords, private key contents, or SSH agent data.

## Development

```sh
make build      # build bin/gate
make test       # run tests with the race detector
make vet        # run go vet
make check      # test + vet
make coverage   # write coverage.out and show package coverage
make install    # install into GOBIN/GOPATH/bin
make clean      # remove generated artifacts
```

A local installation smoke test can be run without changing the normal Go bin
directory:

```sh
GOBIN="$(mktemp -d)" go install ./cmd/gate
```

## Contributing

Bug reports, feature requests, and pull requests are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the workflow and
[SECURITY.md](SECURITY.md) for the security model and how to report
vulnerabilities. The original design document is kept in
[docs/PLAN.md](docs/PLAN.md); maintainer release steps are in
[RELEASING.md](RELEASING.md).

## License

[MIT](LICENSE) © WariKoda

## Current limitations

- Unix terminals only; Windows is not supported.
- Conditional `Match` blocks are not connection targets. Only `Match all` and
  `Match host` contribute to the displayed settings.
- Stargate is an SSH launcher, not an embedded terminal or SFTP client.
