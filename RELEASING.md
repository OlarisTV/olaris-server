## How to release a new build of Olaris

* Update dependencies
  * Merge latest ffmpeg release
    * `git fetch origin`
    * Find latest release in `git tag`
    * Merge latest release, e.g. `git merge n4.2.2` and fix any merge conflicts that may arise
    * TODO(Leon Handreke): Maybe we should actually do this a different way? Maintain a patchset that we always rebase to keep a better overview of the changes
    * `git push olaris`
    * Wait for CI to build ffmpeg as olaris-server pulls it in through its build process

  * Merge latest videojs, videojs-http-streaming release
    * git rebase -i [the latest stable release], pick all but the „Generate dist files“ commit
    * npm install && npm run-script build && git reset HEAD && git add dist && git commit -m "Generate dist files"
    * git push –force gitlab-olaris
    * yarn upgrade @videojs/http-streaming video.js


## Release procedures for individual packages

### olaris-react

olaris-react goes first because its release gets built into the olaris-server binary.
  * Update version number in `package.json`
  * git commit with version bump
  * `git tag v0.3.0`
  * `git push --tags`
  * Wait for the new version to be build on CI because olaris-server `make ready-ci` later downloads this version

### olaris-server

  git tag v0.3.0
  git push --tags
  # Download builds of our modified ffmpeg and olaris-react
  make ready-ci
  # This will place a zip ready for release in build/
  GOOS=linux GOARCH=amd64 make dist
  # Update the docker image
  make docker-build docker-tag docker-push
  # Update the from-ci image
  make docker-from-ci-build-tag-push

### Updatete public releases

To avoid any trouble with your local machine, it's best to use the CI builds for doing the actual release. Download the `dist-linux-amd64` file from Gitlab and extract it. Upload the zip `olaris-linux-amd64-v0.3.0.zip` file to the `olaris-release/` directory on [our Google Cloud Storage](https://console.cloud.google.com/storage/browser/bysh-chef-files/?project=electric-charge-161111). Replace the `olaris-linux-amd64` binary with the unpacked binary so that the Bytesized installer can pull it directly. Update the Blog if necessary (even for a point release, we usually update the link from the latest release blogpost).
