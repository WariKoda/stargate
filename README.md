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
go install github.com/nibra180/stargate-tui/cmd/stargate@latest
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
stargate
```

Use a different config file:

```sh
stargate --config /path/to/ssh_config
```

The config path is selected in this order:

1. `--config PATH`
2. `STARGATE_CONFIG`
3. `~/.ssh/config`

### Controls

| Action | Keyboard | Mouse |
|---|---|---|
| Navigate | `↑`/`↓` or `j`/`k` | Scroll wheel |
| Filter | `/`, then type | — |
| Apply filter | `Enter` | — |
| Clear/cancel filter | `Esc` | — |
| Connect | `Enter` | Left-click a host |
| Toggle extended help | `?` | — |
| Quit | `q` or `Ctrl+C` | — |

When an SSH session ends, Stargate restores the TUI and remains available for
the next connection.

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
make build      # build bin/stargate
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
GOBIN="$(mktemp -d)" go install ./cmd/stargate
```

## Release checklist

1. Run `make check` and the PTY smoke tests.
2. Commit all release changes.
3. Create a semantic version tag, for example `v0.1.0`.
4. Push the commit and tag to GitHub. Make the repository public, or document
   the required Git authentication and `GOPRIVATE` settings.
5. Verify:

   ```sh
   go install github.com/nibra180/stargate-tui/cmd/stargate@v0.1.0
   ```

No release tag is created automatically by the build.

## Current limitations

- Unix terminals only; Windows is not supported.
- Conditional `Match` blocks are not connection targets; `Match exec` is
  rejected.
- Stargate is an SSH launcher, not an embedded terminal or SFTP client.
