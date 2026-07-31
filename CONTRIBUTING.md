# Contributing to Stargate

Thanks for your interest. Bug reports, documentation fixes, and pull requests
are all welcome.

## Before you start

Stargate is deliberately small: it is an SSH **launcher**, not a terminal
emulator, not an SFTP client, and not an SSH implementation. The following are
explicitly out of scope, and pull requests adding them will likely be declined:

- Windows support (Unix terminals only)
- `Match exec` evaluation and conditional config blocks
- Grouping or tags (not a native SSH config concept)
- SFTP, file transfer, or a port-forwarding UI
- An embedded PTY or a reimplementation of the SSH client

See [docs/PLAN.md](docs/PLAN.md) for the original design document explaining why
these boundaries exist. It is written in German; the decision table in section 0
is the quickest summary.

For anything larger than a bug fix, please open an issue first so we can agree
on the approach before you spend time on it.

## Development setup

You need Go (the version in `go.mod` or newer), an OpenSSH client in `$PATH`,
and a Unix-like system.

```sh
git clone https://github.com/nibra180/stargate-tui
cd stargate-tui
make check
```

Available targets:

```sh
make build      # build bin/gate
make test       # run tests with the race detector
make vet        # run go vet
make check      # test + vet
make coverage   # write coverage.out and show package coverage
make install    # install into GOBIN/GOPATH/bin
make clean      # remove generated artifacts
```

To try your build without touching your normal Go bin directory:

```sh
GOBIN="$(mktemp -d)" go install ./cmd/gate
```

## Making changes

- **Format with `gofmt`.** CI fails on unformatted files.
- **`make check` must pass** before you open a pull request. CI runs `gofmt`,
  `go vet`, `go test -race`, and a build on both Linux and macOS.
- **Add tests for behavior changes.** Coverage sits around 82%; please don't
  regress it. Config parsing and the SSH executor are the parts most worth
  testing thoroughly.
- **Never put real hostnames in tests or docs.** Use `example.com`,
  `example.org`, or documentation IP ranges (`192.0.2.0/24`, `2001:db8::/32`).
- **Keep the layers separate.** Parsing lives in `internal/config`, rendering
  and input handling in `internal/tui`, command construction in `internal/ssh`.
  Business logic does not belong in the UI layer.
- **Changes to `internal/ssh` deserve extra care.** That package builds the
  command line handed to the native client; read [SECURITY.md](SECURITY.md)
  before changing how arguments are constructed, quoted, or validated.

Changes to the TUI can't be fully covered by unit tests. Please also do a manual
pass: launch `gate`, filter, connect to a host, and confirm the terminal is
restored cleanly when the session ends.

## Commit messages and pull requests

Write commit subjects in the imperative mood ("add connectivity checks", not
"added"). Conventional-commit prefixes such as `feat:` and `fix:` are used in
this repository but are not enforced.

In your pull request, describe what changed and why, note anything you tested
manually, and link any related issue.

## Reporting bugs

Open an issue using the bug report template. Please include your OS, terminal,
Stargate version, and OpenSSH version, plus a minimal `ssh_config` snippet that
reproduces the problem — with real hostnames replaced.

For security vulnerabilities, do not open a public issue; follow
[SECURITY.md](SECURITY.md) instead.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
