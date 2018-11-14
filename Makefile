.DEFAULT_GOAL := build-local

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOGET=$(GOCMD) get
GOGENERATE=$(GOCMD) generate
BIN_LOC=build
BINARY_NAME=olaris
GODEP=dep
CMD_SERVER_PATH=cmd/olaris/main.go
REACT_REPO=https://gitlab.com/olaris/olaris-react.git
SRC_PATH=gitlab.com/olaris/olaris-server
LDFLAGS=-ldflags "-X $(SRC_PATH)/helpers.GitCommit=$(GIT_REV)"
GIT_REV := $(shell git rev-list -1 HEAD)
REACT_BUILD_DIR=./app/build
IDENTIFIER=$(BINARY_NAME)-$(GOOS)-$(GOARCH)

all: generate

.PHONY: ready-ci
ready-ci: download-olaris-react download-ffmpeg

.PHONY: download-ffmpeg
download-ffmpeg:
	curl -L 'https://gitlab.com/api/v4/projects/olaris%2Fffmpeg/jobs/artifacts/master/download?job=compile' > ffmpeg/executable/build.zip
	unzip -o ffmpeg/executable/build.zip -d ffmpeg/executable/
	cp ffmpeg/executable/ffmpeg-static/bin/ffmpeg ffmpeg/executable/build/
	cp ffmpeg/executable/ffmpeg-static/bin/ffprobe ffmpeg/executable/build/
	rm ffmpeg/executable/build.zip
	rm -rf ffmpeg/executable/ffmpeg-static
	make generate

.PHONY: download-olaris-react
download-olaris-react:
	curl -L 'https://gitlab.com/api/v4/projects/olaris%2Folaris-react/jobs/artifacts/develop/download?job=compile' > react/static.zip
	unzip -o react/static.zip -d react/
	rm react/static.zip
	make generate

.PHONY: build-olaris-react
build-olaris-react:
	if [ ! -d "./builds/olaris-react" ]; then mkdir -p builds && cd builds && git clone $(REACT_REPO) olaris-react; fi
	cd builds/olaris-react && git fetch --all && git reset --hard origin/develop && yarn install && yarn build
	cp -r builds/olaris-react/build ./react/
	make generate

.PHONY: build
build: generate
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -o $(BIN_LOC)/$(IDENTIFIER) $(LDFLAGS) -v $(CMD_SERVER_PATH)

.PHONY: build-local
build-local: generate
	$(GOBUILD) -o $(BIN_LOC)/$(BINARY_NAME) $(LDFLAGS) -v $(CMD_SERVER_PATH)

build-docker:
	docker build -t olaris-server .

.PHONY: crossbuild
crossbuild:
	mkdir -p $(BIN_LOC)
	make build FLAGS="$(BIN_LOC)/$(IDENTIFIER)"

.PHONY: test
test:
	$(GOTEST) -v ./...

.PHONY: vet
vet:
	$(GOVET) -v ./...

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf ./builds

.PHONY: deps
deps:
	$(GODEP) ensure

.PHONY: generate
generate:
	$(GOGENERATE) -v ./...

.PHONY: run
run: all
	$(GOCMD) $(CMD_SERVER_PATH)

.PHONY: build-all
build-all:
	make crossbuild GOOS=linux GOARCH=arm
	make crossbuild GOOS=linux GOARCH=386
	make crossbuild GOOS=linux GOARCH=arm64
	make crossbuild GOOS=linux GOARCH=amd64

.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs
