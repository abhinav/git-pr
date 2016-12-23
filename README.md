This repository is intended to contain tools to make my GitHub workflow easier.
Documentation and tests are lacking at this time.

Installation
============

    go get github.com/abhinav/git-fu/cmd/...

Commands
========

The following commands have been implemented so far.

`git-pr`
--------

Only one subcommand at this time: `land`. It squash-merges the pull request
associated with a specific branch (defaults to the current branch) and deletes
the associated remote branch. The commit message for the squashed commit is
picked up from the pull request description, with support for editing it before
merging.

    git pr land
    git pr land mybranch

Code organization
=================

-   Each standalone command will go under cmd/
-   Shared library functions will stay at the top-level

Stability
=========

Pre-alpha. Extremely unstable.
