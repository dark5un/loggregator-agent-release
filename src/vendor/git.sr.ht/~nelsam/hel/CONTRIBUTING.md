Contributing to hel
-------------------

## Versioning

hel will always remain at v0 to avoid `go module` issues with v1 and above.
This means that our version number follows the pattern `v0.{major}.{minor}`.
This also means that "minor" versions contain both new features and bug fixes.

## Tests

hel development generally follows a test-driven approach. Write a test that
fails against current `main`, first. This both proves that there is an issue and
proves that the test fails in the expected way.

## Making Your PR

hel makes use of sourcehut's mailing list patchsets. Feel free to prepare a
patchset through the UI if you want. Just click on the `Prepare a patchset`
button.

If you prefer to use the CLI, [set up git send-email](https://git-send-email.io)
and send your patchsets to `~nelsam/hel@lists.sr.ht`.

For convenience, hel provides `earthly +send-patch` to send a patchset from
`HEAD` to `origin/main` (by default). Note that this does not support
`--annotate` because it can't open an editor. If you want to annotate your
patches, use `git send-email` directly.

### Backports

We generally don't backport to old versions. If you _absolutely need_ a feature
or bugfix backported, we can discuss it - but you are probably better off using
the `replace` directive in your `go.mod` to point to a fork than trying to lock
your `go.mod` to an old official version.
