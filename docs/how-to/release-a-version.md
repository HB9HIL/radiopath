# Release a version

Releases are cut by the maintainer from `main`.

1. Make sure `main` is green in CI and `CHANGELOG.md` has every change since the last release under
   `## [Unreleased]`.
2. Pick the version by [Semantic Versioning](https://semver.org): a fix is a patch, a new feature a minor, an
   incompatible change (API, configuration) a major.
3. Rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD` and add a new empty `## [Unreleased]` above it.
4. Commit with `chore: release X.Y.Z` and push.
5. Tag and push the tag:

   ```sh
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

The tag run builds and tests like any other run, then publishes the image as `ghcr.io/radiopath/radiopath:X.Y.Z`
and `:latest` and creates the GitHub release with the changelog section of that version as notes and the Linux
binaries attached. It fails if the changelog has no section for the version.
