# Releasing

Maintainer notes. Nothing here is needed to use or contribute to Stargate.

Stargate is distributed through the Go module proxy, so a release is a signed
git tag — no artifacts are built or uploaded, and no tag is created
automatically by the build.

## Checklist

1. Confirm CI is green on `main`.
2. Run `make check` locally, plus a manual PTY smoke test: launch `gate`,
   connect to a host, exit the session, and confirm the terminal is restored.
3. Confirm the working tree is clean and all release changes are committed.
4. Create an annotated semantic version tag:

   ```sh
   git tag -a v0.1.0 -m "v0.1.0"
   ```

5. Push the commit and the tag:

   ```sh
   git push origin main
   git push origin v0.1.0
   ```

6. Verify the module is installable from the proxy. Allow a minute for the proxy
   to pick up the tag:

   ```sh
   GOBIN="$(mktemp -d)" go install github.com/nibra180/stargate-tui/cmd/gate@v0.1.0
   ```

7. Create the GitHub release from the tag and summarize the user-visible
   changes.

## Versioning

Stargate follows [Semantic Versioning](https://semver.org/). While the version
is below `v1.0.0`, the keybindings, CLI flags, and output format may still
change in a minor release; such changes are called out in the release notes.

## Repository visibility

`go install ...@latest` requires the module path to be publicly resolvable. If
the repository is private, consumers must configure Git authentication and
`GOPRIVATE=github.com/nibra180/*` themselves — document this in the README if
that stays the case.
