# Release process

To create a new release:

1. Tag your commit with a semver tag prefixed with 'v':
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The GitHub Actions workflow will automatically:
    - Build binaries for all supported platforms (Linux, macOS, Windows)
    - Extract the entries for the tagged version from `CHANGELOG.md`
    - Create a GitHub release
    - Upload the built artifacts

The release notes will include:
- Version and release date
- Installation instructions
- The `CHANGELOG.md` entries for the tagged version
- Link to the full changelog comparing with the previous tag

The entries are extracted by the [since](https://github.com/release-tools/since)
action and passed to GoReleaser via `--release-notes`, so the release body
matches the changelog exactly. The release job fails if the most recent entry in
`CHANGELOG.md` is not for the tag being released — run `since project release`
(or update the changelog by hand) before tagging.
