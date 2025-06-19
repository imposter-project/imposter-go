# Release process

To create a new release:

1. Tag your commit with a semver tag prefixed with 'v':
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The GitHub Actions workflow will automatically:
    - Build binaries for all supported platforms (Linux, macOS, Windows)
    - Generate a changelog from commit messages
    - Create a GitHub release
    - Upload the built artifacts

The release notes will include:
- Version and release date
- Automatically generated changelog (excluding docs, test, ci, and chore commits)
- Installation instructions
- Link to the full changelog comparing with the previous tag
