include .bingo/Variables.mk

PREFIX            ?= $(shell pwd)
FILES_TO_FMT      ?= $(shell find . -path ./vendor -prune -o -name '*.go' -print)

DOCKER_IMAGE_REPO ?= quay.io/thanos/thanosbench
DOCKER_IMAGE_TAG  ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))-$(shell date +%Y-%m-%d)-$(shell git rev-parse --short HEAD)

# Ensure everything works even if GOPATH is not set, which is often the case.
# Default to standard GOPATH.
GOPATH            ?= $(HOME)/go

TMP_GOPATH        ?= /tmp/thanos-go
GOBIN             ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO111MODULE       ?= on
export GO111MODULE
GOPROXY           ?= https://proxy.golang.org
export GOPROXY

GIT               ?= $(shell which git)

# Support gsed on OSX (installed via brew), falling back to sed. On Linux
# systems gsed won't be installed, so will use sed as expected.
SED ?= $(shell which gsed 2>/dev/null || which sed)

define require_clean_work_tree
	@git update-index -q --ignore-submodules --refresh

    @if ! git diff-files --quiet --ignore-submodules --; then \
        echo >&2 "cannot $1: you have unstaged changes."; \
        git diff-files --name-status -r --ignore-submodules -- >&2; \
        echo >&2 "Please commit or stash them."; \
        exit 1; \
    fi

    @if ! git diff-index --cached --quiet HEAD --ignore-submodules --; then \
        echo >&2 "cannot $1: your index contains uncommitted changes."; \
        git diff-index --cached --name-status -r --ignore-submodules HEAD -- >&2; \
        echo >&2 "Please commit or stash them."; \
        exit 1; \
    fi

endef

.PHONY: all
all: format build

.PHONY: build
build: check-git deps $(PROMU)
	@echo ">> building binaries $(GOBIN)"
	@$(PROMU) build --prefix $(PREFIX)

# crossbuild builds all binaries for all platforms.
.PHONY: crossbuild
crossbuild: $(PROMU)
	@echo ">> crossbuilding all binaries"
	$(PROMU) crossbuild -v

.PHONY: gen
gen:
	@echo ">> generating benchmarks configs"
	@rm -rf benchmarks/**/manifests
	@go run benchmarks/base/main.go generate
	@go run benchmarks/lts/main.go generate --tag=v0.8.1
	@go run benchmarks/remote-read/chunkiter/main.go generate
	@go run benchmarks/remote-read/streamedchunks/main.go generate

.PHONY: promu
promu: $(PROMU)

.PHONY: tarballs-release
tarballs-release: $(PROMU)
	@echo ">> Publishing tarballs"
	$(PROMU) crossbuild -v tarballs
	$(PROMU) checksum -v .tarballs
	$(PROMU) release -v .tarballs

# deps ensures fresh go.mod and go.sum.
.PHONY: deps
deps:
	@go mod tidy
	@go mod verify

# docker builds docker with no tag.
.PHONY: docker
docker: build
	@echo ">> building docker image 'thanosbench'"
	@docker build -t "thanosbench" .

# docker-push pushes docker image build under `thanos` to "$(DOCKER_IMAGE_REPO):$(DOCKER_IMAGE_TAG)"
.PHONY: docker-push
docker-push:
	@echo ">> pushing thanosbench image as $(DOCKER_IMAGE_REPO):$(DOCKER_IMAGE_TAG)"
	@docker tag "thanosbench" "$(DOCKER_IMAGE_REPO):$(DOCKER_IMAGE_TAG)"
	@docker push "$(DOCKER_IMAGE_REPO):$(DOCKER_IMAGE_TAG)"

# format formats the code (including imports format).
.PHONY: format
format: $(GOIMPORTS)
	@echo ">> formatting code"
	@$(GOIMPORTS) -w $(FILES_TO_FMT)

.PHONY: check-git
check-git:
ifneq ($(GIT),)
	@test -x $(GIT) || (echo >&2 "No git executable binary found at $(GIT)."; exit 1)
else
	@echo >&2 "No git binary found."; exit 1
endif

.PHONY: docs
docs: ## Regenerates flags in docs for all thanos commands.
docs: $(EMBEDMD) build
	@EMBEDMD_BIN="$(EMBEDMD)" SED_BIN="$(SED)" THANOSBENCH_BIN="./thanosbench" scripts/genflagdocs.sh
	@find . -type f -name "*.md" | SED_BIN="$(SED)" xargs scripts/cleanup-white-noise.sh

.PHONY: lint
# PROTIP:
# Add
#      --cpu-profile-path string   Path to CPU profile output file
#      --mem-profile-path string   Path to memory profile output file
#
# to debug big allocations during linting.
lint: check-git $(GOLANGCI_LINT) $(MISSPELL)
	@echo ">> linting all of the Go files GOGC=${GOGC}"
	@$(GOLANGCI_LINT) run --enable goimports --enable goconst --skip-dirs vendor
	@echo ">> detecting misspells"
	@find . -type f | grep -v vendor/ | grep -vE '\./\..*' | xargs $(MISSPELL) -error

# checks Go code comments if they have trailing period (excludes protobuffers and vendor files).
# Comments with more than 3 spaces at beginning are omitted from the check, example: '//    - foo'.
.PHONY: check-comments
check-comments:
	@printf ">> checking Go comments trailing periods\n\n\n"
	@./scripts/build-check-comments.sh

.PHONY: test
test:
	@echo ">> running tests"
	@go test ./...
