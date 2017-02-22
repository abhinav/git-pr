Releases
========

v0.2.1 (2017-02-21)
-------------------

-   Fix bug in `git pr rebase` where commits from old rebased bases were
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
