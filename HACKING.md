# Hacking
This document is an overview of how to build and run Olaris, as well as how to get your changes merged back upstream.

[[_TOC_]]

## Overview
The general idea is to check out a copy of the source code, compile it into a binary, and run it.

Once you're able to do that, you can start making changes to the code and test them. If you want, you can go and look at the [outstanding issues on Gitlab](https://gitlab.com/olaris/olaris-server/issues/). There are tags for issues specifically targetted to new contributors [here](https://gitlab.com/olaris/olaris-server/issues?scope=all&utf8=%E2%9C%93&state=opened&label_name[]=Good%20first%20issue)

Once you've made a change in the code that fixes an issue (and tested it), the steps to get it accepted are [forking the repo on Gitlab](https://docs.gitlab.com/ee/user/project/repository/forking_workflow.html#creating-a-fork), pushing your change to a feature branch, and then submitting a [merge request](https://docs.gitlab.com/ee/user/project/merge_requests/creating_merge_requests.html). 

### What's in this repo
The server handles the backend - scanning your media library, querying metadata from [The Movie DB](https://www.themoviedb.org/), building the database - as well as the frontend - serving the webapp via https, reading from the database and streaming videos from the backend to the frontend.

### What's not in this repo

The client (playback) software for any device that is not a modern web browser. ie. Android, iOS, Roku etc. There is an iOS client under development [here](https://gitlab.com/olaris/olaris-ios) but it's still in alpha state and a source-only release at this point.

### Supported platforms
 * Linux - these instructions were written for Arch but any distribution should work
 * Golang, v1.13+ recommended. You can install this through your distribution repos but the latest version is always available at https://golang.org
 * A custom version of ffmpeg. Binaries are provided for amd64 but you will need to build this yourself on other architectures
 * Git is required if you want to contribute changes back and optionally to obtain a copy of the code
 * Make

# Building and running

## Getting the code
You can authenticate to Gitlab using username/password or ssh. Setting up ssh keys is beyond the scope of this document but there are instructions [here](https://docs.gitlab.com/ee/ssh/)

    git clone git@gitlab.com:olaris/olaris-server.git

## Install the toolchain

  * Install the Go toolchain, either from your distro repos or directly from the [Go project](https://golang.org/dl)
  * Install make
  * Optionally, build our custom [ffmpeg](https://gitlab.com/olaris/ffmpeg) if you want to actually transcode and playback video and you are not on a Linux amd64 system.

## Build dependencies

There is a makefile that can handle various project tasks.

  * Run `make deps` to install some third party tools
  * Run `make download-olaris-react` to grab the latest build of the web frontend for Olaris.
  * Run `make download-ffmpeg` to download the custom build of ffmpeg required for Olaris to function. This will only work if you are on an `amd64` Linux machine. If you are on another platform, you will have to [build it yourself](https://gitlab.org/olaris/olaris-ffmpeg).

## Build olaris

  * `make build-local` to build a binary for your local platform. The binary will be placed in `build/olaris`.

## Running the server

By default, olaris-server will open an existing database or create a new one and start listening for web connections on port 8080. For development you may want to override the defaults, for example to run against a copy of your primary olaris db instead of the real one.

You can run the compiled binary as follows:
    `build/olaris --config_dir ~/olaris_dev_cfg/`

# Merging your changes upstream

So you've fixed a bug or added a new feature and now you want to merge your changes back to the main project so everyone can benefit.

## Before you start

### Ensure your code is formatted correctly
  * `go fmt -w <filename>` for each file you have modified

### Ensure tests are passing
You can test that the CI/CD pipeline will run locally as follows:
  *  `make vet`  runs a linter on the code
  *  `make test` to run the test suite ensure all tests still pass, including any new ones you've added

Adding tests is not required but is encouraged; making sure existing tests do not break is mandatory.

## Fork the repo and push your change
Once you've created an account on Gitlab, [forking the repo](https://docs.gitlab.com/ee/user/project/repository/forking_workflow.html#creating-a-fork) creates your own copy of the repo that you can push your changes to.

One possible git flow for pushing your changes is as follows:

  * `git remote set-url origin git@gitlab.com:<your username>/olaris-server.git` to point your local git client at your personal Gitlab repo instead of the master Olaris repo
  * `git checkout -b some-descriptive-name` to create a local feature branch
  * `git add <filenames>` to stage each file you've modified
  * `git commit -m 'some description of your work'` to commit your staged changes with a relevant message
  * `git push --set-upstream origin <your branch name>` to push your local branch to a branch on Gitlab with the same name

## Submit a merge request
Once you have created a feature branch and pushed it to your fork of the repo, you can submit a [merge request](https://docs.gitlab.com/ee/user/project/merge_requests/creating_merge_requests.html) to the Olaris project. It never hurts to paste a link to your MR in the Discord channel but please be patient if nobody is able to look at it right away.

## Tips

After you've opened a merge request, you may see that the CI/CD pipeline is stalled or has failed due to not having a Runner. Gitlab offers a free runner to open source projects, but you need to enable it from your project page. Go to `Settings > CI/CD > Runners` then click `Expand` and finally `Enable shared Runners`
