![Olaris server header](https://i.imgur.com/ewz5TAN.png)

## `This is all pre-release code, continue at your own peril.`

## What is Olaris?

Olaris is an open-source, community driven, media manager and transcoding server. The main interface is the [olaris-react](https://gitlab.com/olaris/olaris-react) project although in due time we hope to support multiple clients / applications.

Our core values are:

### Community driven development
We want Olaris to be a community project which means we will heavily prioritise features based on our user feedback.

### Focus on one feature at a time
We will work on features until they are perfect (or as close to it as possible). We would rather have a product where three features work really well than a product with 30 unfinished features.

This does not mean we won't work in parallel, it simply means we will not start anything new until we are happy the new feature works to a high standard.

### Our users are not our product
We don't want to collect metadata, we don't want to sell metadata your data is yours and yours alone.

### Singular Focus: Video.
Our sole focus is on video and video alone, anything that does not meet this requirement will not be considered. This means for example we will never add music support due to different approach that would be required throughout the application. 

### Open-source
Everything we build should be open-source. We feel strongly that more can be achieved with free open-source software. That's why were are aiming to be and to remain open-source instead of open-core where certain features are locked behind a paywall.


## Setting up a build environment

### Build dependencies
  * Install the [Go toolchain](https://golang.org)
  * Install some third party tools
	  * go get github.com/jteeuwen/go-bindata/...
	  * go get github.com/elazarl/go-bindata-assetfs/...
	  * go get github.com/maxbrunsfeld/counterfeiter
  * Build our custom [ffmpeg](https://gitlab.com/olaris/ffmpeg) if you want to actually transcode and playback video and you are not on a Linux amd64 system.

### Make commands

  First run 
  `make download-olaris-react` this will grab the latest Olaris react build from the CI and insert it into olaris-server.
  Optionally if you are on Linux amd64 you can use `make download-ffmpeg` to do the same for ffmpeg. If not you will have to build it yourself.

  Once this is done you can use some of the following commands:
 
  `make run` to run a build on your local platform.

  `make build` to build a binary for your local platform.

  `make build-with-react` to build and pull in the latest web-interface this requires the build toolchain for [olaris-react](https://gitlab.com/olaris/olaris-react).

  `make list` for all make options
