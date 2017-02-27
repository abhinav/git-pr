Releases
========

v0.4.1 (2017-02-27)
-------------------

-   Fix auto-released Darwin binaries.


v0.4.0 (2017-02-22)
-------------------

-   The `--onto` parameter for `git pr rebase` is now optional. If not
    specified, the current branch is not rebased but all its dependents are
    rebased on its updated head.


v0.3.0 (2017-02-22)
-------------------

-   `git pr rebase` now resets local branches when their remotes get updated
    but only if their heads matched the remotes at the time of rebase.


v0.2.2 (2017-02-22)
-------------------

-   `git pr rebase`: Use a single `git push` command rather than multiple when
    updating remote refs. This should significantly speed things up.
-   Fixed bug in `git pr rebase` that would cause the base branches for all PRs
    to be changed to the given base branch rather than just the first level.


v0.2.1 (2017-02-21)
-------------------

-   Fixed bug in `git pr rebase` where commits from old rebased bases were
    leaking into higher level branches.
-   `git pr rebase` now stages all rebase changes locally on temporary branches
    before pushing them to the remotes. This way, no actual changes are made
    until we verify locally that everything fits.


v0.2.0 (2017-02-21)
-------------------

-   Added `rebase` subcommand to `git pr`.


v0.1.0 (2017-02-20)
-------------------

-   Initial release.
