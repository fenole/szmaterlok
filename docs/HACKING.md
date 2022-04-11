# Hacking Guide

In [fenole/szmaterlok](https://github.com/fenole/szmaterlok) project we applied
[GitHub flow](https://docs.github.com/en/get-started/quickstart/github-flow) for
our main workflow.

You don't really need to be a part of [fenole](https://github.com/fenole)
organisation to start hacking on [szmaterlok](https://github.com/fenole).

## Before you start

In order to start working with **szmaterlok**, you will need couple things setup
before.

1. [SSH](https://docs.github.com/en/authentication/connecting-to-github-with-ssh)
   and [git](https://docs.github.com/en/get-started/quickstart/set-up-git)
   setup.
2. Unix environment (can be [WSL](https://docs.microsoft.com/en-us/windows/wsl/)
   if you're Windows user).
3. Some code editor ([Visual Studio Code](https://code.visualstudio.com/) is
   great for beginners).
4. [zsh](https://repology.org/project/zsh/versions)(pre-installed on macOS)
   shell, [go](https://go.dev/dl/) compiler and
   [deno](https://deno.land/#installation) toolchain.

If you fulfil all of the above requirements you can proceed further.

## Git workflow

You need your own fork of **fenole/szmaterlok** repository pinned to your github
account. You can read
[here](https://docs.github.com/en/get-started/quickstart/fork-a-repo) how to
fork repositories on GitHub.

Then you can clone this repository somewhere around your hardrive.

    $ git clone git@github.com:fenole/szmaterlok.git

Now you can enter freshly cloned directory and setup your fork.

    $ cd fenole
    $ git remote add fork git@github.com:<your_username>/szmaterlok.git

Make sure to replace `<your_username>` with your actual username. So, for
example, if your username is `pawel`, the above command will be looking
following way.

    $ git remote add fork git@github.com:pawel/szmaterlok.git

Now you can create new branch. We're using kebab-case for naming our branches at
**szmaterlok**. Branch name should describe briefly what's your intention.

    $ git switch -c <name_of_branch>

The real life scenario may look like below.

    $ git switch -c refactor-some-very-important-code

Not it's time to make soma changes and commit your code. Please, follow git
commit messages convention described at the end of this document.

When your changes are ready, you can push your branch to your remote forked
repository and open merge request.

    $ git push fork <name_of_branch>

If you want to start working on another feature: you can switch back to main
branch and fetch upstream changes.

    $ git switch main
    $ git fetch --all
    $ git pull origin main

Now you can create another branch and repeat above steps. Enjoy your hacking!

## Code style

Use `./make.zsh` script to format your code before you'll open merge request.
We're using `go fmt` for formatting go code and `deno fmt` for formatting
javascript and markdown files.

Execute below command in the root of the repository to format all of the files.

    $ ./make.zsh fmt

## Building and running project

`make.zsh` scripts contains multiple functions for maintenance and development
of project.

Here's how to run it.

    $ ./make.zsh <function>

Where function can be one of following: `help`, `fmt`, `go:build` and others.

Use `help` command to discover all options.

Below is brief description of the most interesting recipies.

### go:build

Build binaries, which you can run on your local machine.

    $ ./make.zsh go:build
    $ ./szmaterlok

### go:run

Build and run chat.

### go:watch

Build, run and rebuild chat every time you introduce new changes in some source
file from repository. Very useful when working with UI changes.

### go:test

Execute go unit tests with verbose output.

## Commit messages

Please use below template for git commit messages. Every commit should follow
this rule.

```
# feat: add hat wobble
# ^--^  ^------------^
# |     |
# |     +-> Summary in present tense.
# |
# +-------> Type: feat, fix, docs, style, refactor, perf, test, build, ci, chore or revert.
#
# Types:
# * feat        A new feature
# * fix         A bug fix
# * docs        Documentation only changes
# * style       Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
# * refactor    A code change that neither fixes a bug nor adds a feature
# * perf        A code change that improves performance
# * test        Adding missing tests or correcting existing tests
# * build       Changes that affect the build system or external dependencies (example scopes: gulp, broccoli, npm)
# * ci          Changes to our CI configuration files and scripts (example scopes: Travis, Circle, BrowserStack, SauceLabs)
# * chore       Other changes that don't modify src or test files
# * revert      Reverts a previous commit
```

**Optional** (but handy): save content of above snippet as `$HOME/.gitmessage`
and execute below command to use it as template for all of your git messages.

    $ git config --global commit.template ~/.gitmessage
