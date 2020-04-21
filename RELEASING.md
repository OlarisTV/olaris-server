## How to release a new build of Olaris

* Update dependencies
  * Merge latest ffmpeg release
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

