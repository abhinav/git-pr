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

Only one subcommand at this time: `land`. This does a few things:

-   Squash-merges a specific pull request, defaulting to the pull request made
    with the current branch
-   Allows editing the commit message for the squash commit, defaulting to the
    PR title and body for the commit message
-   Pulls the merge base
-   Performs post-merge cleanup like deleting local and remote branches
-   Rebases pull requests that depend on the merged pull request against its
    base and changes their merge base on GitHub to that branch

    git pr land
    git pr land mybranch

In pictures, given the layout,

         o---o feature2
        /
    o--o master
        \
         o---o feature1
              \
               o---o feature3

Running,

    $ git checkout feature2
    $ git pr land

The resulting layout will look like so,

    o--o--o master'
        \
         o---o feature1
              \
               o---o feature3

      master' = master + feature2

Afterwards, running the following,

    $ git checkout feature1
    $ git pr land

Will result in,

    o--o--o--o master''
              \
               o---o feature3

      master'' = master' + feature1

Rebasing dependent pull requests is intended to work across multiple levels.
PRs that depend on rebased PRs will also be rebased. For example, given,

    o---o master
         \
          o feature1
           \
            o--o--o feature2
             \
              o--o feature3

Landing feature1 will result in,

    o---o---o master'
             \
              o--o--o feature2
               \
                o--o feature3

      master' = master + feature1

Code organization
=================

-   Each standalone command will go under cmd/
-   Shared library functions will stay at the top-level

Stability
=========

Pre-alpha. Extremely unstable. Barely tested.
