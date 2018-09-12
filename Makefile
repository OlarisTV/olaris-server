.DEFAULT_GOAL := build_local

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOGET=$(GOCMD) get
GOGENERATE=$(GOCMD) generate
BIN_LOC=build
BINARY_NAME=olaris-server
GODEP=dep
CMD_SERVER_PATH=cmd/olaris-server/main.go
REACT_REPO=https://gitlab.com/olaris/olaris-react.git
SRC_PATH=gitlab.com/olaris/olaris-server
LDFLAGS=-ldflags "-X $(SRC_PATH)/helpers.GitCommit=$(GIT_REV)"
GIT_REV := $(shell git rev-list -1 HEAD)
REACT_BUILD_DIR=./app/build
IDENTIFIER=$(BINARY_NAME)-$(GOOS)-$(GOARCH)

all: generate

.PHONY: ready-ci
ready-ci:
	curl -L 'https://gitlab.com/api/v4/projects/olaris%2Folaris-react/jobs/artifacts/develop/download?job=compile' > react/static.zip
	unzip react/static.zip -d react/
	make generate

.PHONY: update-react
update-react:
	if [ ! -d "./builds" ]; then git clone $(REACT_REPO) builds; fi
	cd builds && git fetch --all && git reset --hard origin/develop && yarn install && yarn build
	cp -r builds/build ./react/

.PHONY: build
build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -o $(BIN_LOC)/$(IDENTIFIER) $(LDFLAGS) -v $(CMD_SERVER_PATH)

.PHONY: build_local
build_local:
	$(GOBUILD) -o $(BIN_LOC)/$(BINARY_NAME) $(LDFLAGS) -v $(CMD_SERVER_PATH)

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
