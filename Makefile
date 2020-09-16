GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOINSTALL=$(GOCMD) install
BINARY_NAME=tcpmessenger


all: test build
build:  ## Build binary (tcpmessenger) for current os.
		$(GOBUILD) -o $(BINARY_NAME) -v
test:   ## Run tests.
		$(GOTEST) -v ./...
clean:  ## Remove created binary.
		$(GOCLEAN)
		rm -f $(BINARY_NAME)
run:    ## Build and run tcpmessenger (at port 8033).
		$(GOBUILD) -o $(BINARY_NAME) -v
		./$(BINARY_NAME)

.PHONY: help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'