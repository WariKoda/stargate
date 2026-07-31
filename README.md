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

Install the latest published version after the first release has been pushed:

```sh
go install github.com/nibra180/stargate-tui/cmd/gate@latest
```

For a private repository, configure Git authentication and `GOPRIVATE` first.

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
| Filter | Start typing (`/` remains optional) | — |
| Apply filter | `Enter` | — |
| Clear/cancel filter | `Esc` | — |
| Connect | `Enter` | Left-click a host |
| Preview SSH command | `Ctrl+P` | — |
| Copy SSH command | `Ctrl+Y` | — |
| Check selected host | `Ctrl+R` | — |
| Toggle extended help | `?` | — |
| Quit | `Ctrl+C`, or `Esc` when no filter is active | — |

Printable keys, including `j`, `k`, and `q`, are filter input. This keeps direct
typing unambiguous; navigation uses the arrow keys.

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

## SSH config behavior

- `Host` blocks with concrete aliases are listed.
- Multiple concrete aliases on one `Host` line are listed separately.
- Wildcard and negated patterns (`*`, `?`, `!`) are used for effective settings
  but are not shown as connection targets.
- `Include` directives and glob patterns are followed recursively.
- Effective `HostName`, `User`, `Port`, `IdentityFile`, `ProxyJump`, and
  `ProxyCommand` settings use OpenSSH's first-value-wins behavior.
- Hosts using `ProxyJump` or `ProxyCommand` are marked `via Jump-Host`.
- `Match exec` is intentionally unsupported and rejected without executing the
  configured command.

Stargate invokes connections as:

```text
ssh -F <resolved-config-path> -- <selected-alias>
```

This ensures that the native client uses the same config Stargate displayed.
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

## Current limitations

- Unix terminals only; Windows is not supported.
- Conditional `Match` blocks are not connection targets; `Match exec` is
  rejected.
- Stargate is an SSH launcher, not an embedded terminal or SFTP client.
