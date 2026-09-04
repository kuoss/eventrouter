## Development

```
$ make help     # list the available targets
$ make checks   # lint, workflow lint, tests, image build, vulnerability scan
```

Tools (golangci-lint, govulncheck, actionlint) are pinned in the Makefile and
installed under `./bin`, so they do not depend on what is in your `$PATH`.
Pull requests run the same lint and tests; `govulncheck` and the image scan run
on a schedule instead (`.github/workflows/security.yml`), because their result
depends on the vulnerability database rather than on the change.

See [`docs/testing.md`](docs/testing.md) for the shared test fixtures, the
checked-in sample output, and the manual kind-cluster tests `make checks`
does not run.

Opening a pull request assigns it to you
(`.github/workflows/auto-assign.yml`), so the pull request list shows who is
carrying each one. An already assigned pull request is left alone, and bot
authors such as Dependabot are skipped because they cannot be assignees.

## Releasing

Open a pull request that changes `VERSION` to the version you want to release,
for example `0.5.0` (no `v`, and `0.5.0-rc1` for a pre-release). Merging it
is the whole release:

1. `.github/workflows/release.yml` picks up the change to `VERSION`
2. the images are built for every supported platform and pushed to
   `ghcr.io/kuoss/eventrouter:v0.5.0`, plus `:latest` unless it is a
   pre-release
3. `v0.5.0` is tagged and a GitHub release is opened with generated notes

Nothing is tagged or pushed by hand, and the version is reviewable before it
ships. The workflow refuses to run if the tag already exists, so re-merging
something that leaves `VERSION` alone cannot republish a release.

## DCO Sign off

All authors to the project retain copyright to their work. However, to ensure
that they are only submitting work that they have rights to, we are requiring
everyone to acknowledge this by signing their work.

Any copyright notices in this repos should specify the authors as "the contributors".

To sign your work, just add a line like this at the end of your commit message:

```
Signed-off-by: Joe Beda <joe@heptio.com>
```

This can easily be done with the `--signoff` option to `git commit`.

By doing this you state that you can certify the following (from https://developercertificate.org/):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```