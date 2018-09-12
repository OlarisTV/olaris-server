# Olaris Server

## `This is all prelease code, if you somehow found this please move along, this won't work yet :) `

## What is Olaris?

Olaris is an open-source community driven media manager and transcoding server. The main interface is the [olaris-react](https://gitlab.com/olaris/olaris-react) project although in due time we hope to support multiple clients.

Olaris is build using the following values:

### Community driven development
We want Olaris to be a community project which means we will heavily
prioritize features based on user feedback.

### Focus on one feature at a time
We will work on features until they are pefect (or as close to it as possible). We rather have a product where three features work really well then a product with 30 half-assed features.
This does not mean we won't work in parallel it just means that we will not start new features until we are happy the features we deliver actually work.

### Our users are not our product
We don't want to collect metadata, we don't want to sell metadata your data is yours.

### Focus on one thing: Video.
Our focus is on visual media, video. This will be our focus anything that does not meet this requirement won't be worked on. This means for instance that we will never add music support as it just requires a drastically different way of doing things. If we do want to do something with it we will build a new product specifically for this.

### Open-source
Everything we build should be open-source. We feel strongly that more
can be achieved with free open-source software. That's why were are
aiming to be and say open-source instead of open-core.

## Running manually

### Build dependencies
  * Install Go
  * Build and install [ffchunk](https://gitlab.com/olaris/ffchunk)
	* go get github.com/jteeuwen/go-bindata/...
	* go get github.com/elazarl/go-bindata-assetfs/...

### Running

	`make run`

## Building

  `make build` to build for your local platform.

  `build-with-react` to build and pull in the latest web-interface.
