GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOGET=$(GOCMD) get
GOGENERATE=$(GOCMD) generate
BIN_LOC=bin/
BINARY_NAME=$(BIN_LOC)olaris-server
BINARY_UNIX=$(BINARY_NAME)-unix
GODEP=dep
CMD_SERVER_PATH=cmd/olaris-server/main.go
REACT_REPO=https://gitlab.com/olaris/olaris-react.git
SRC_PATH=gitlab.com/olaris/olaris-server
LDFLAGS=-ldflags "-X $(SRC_PATH)/helpers.GitCommit=$(GIT_REV)"
GIT_REV := $(shell git rev-list -1 HEAD)
REACT_BUILD_DIR=./app/build

all: update-react generate test vet

update-react:
	if [ ! -d "./builds" ]; then git clone $(REACT_REPO) builds; fi
	cd builds ; git checkout develop; git pull ; yarn install ; yarn build
	cp -r builds/build ./app/
build:
	$(GOBUILD) -o $(BINARY_NAME) $(LDFLAGS) -v $(CMD_SERVER_PATH)
#build-with-react: update-react generate build
test:
	$(GOTEST) -v ./...
vet:
	$(GOVET) -v ./...
clean:
	$(GOCLEAN)
deps:
	$(GODEP) ensure
generate:
	$(GOGENERATE) -v ./...
run: build
	./$(BINARY_NAME)
build-docker:
	docker build -t olaris-server .
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(CMD_SERVER_PATH)
build-arm6:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GOBUILD) -o $(BINARY_NAME)-arm6 -v $(CMD_SERVER_PATH)
build-arm7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BINARY_NAME)-arm7 -v $(CMD_SERVER_PATH)

build-all: generate build-arm6 build-arm7 build-linux
