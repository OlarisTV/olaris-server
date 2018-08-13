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
CMD_SERVER_PATH=cmd/bytesized-streaming-server/main.go
REACT_REPO=git@gitlab.com:bytesized/bss-react.git

all: test vet build
update-react:
	if [ ! -d "./builds" ]; then git clone $(REACT_REPO) builds; fi
	cd builds ; git checkout develop; git pull ; yarn install ; yarn build
	cp -r builds/build ./app/
build: generate update-react
	$(GOBUILD) -o $(BINARY_NAME) -v $(CMD_SERVER_PATH)
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
build-linux: generate update-react
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(CMD_SERVER_PATH)
build-arm6: generate update-react
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GOBUILD) -o $(BINARY_NAME)-arm6 -v $(CMD_SERVER_PATH)
build-arm7: generate update-react
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o $(BINARY_NAME)-arm7 -v $(CMD_SERVER_PATH)

build-all: build-arm6 build-arm7 build-linux
