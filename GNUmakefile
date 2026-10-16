NAME=zsvirty
BINARY=packer-plugin-${NAME}
PACKER?=packer

check-env:
	@echo "Checking Go environment..."
	@which go >/dev/null || (echo "Error: Go is not installed" && exit 1)
	@echo "Go version: $$(go version)"
	@echo "GOPATH: $${GOPATH}"
	@echo "GOBIN: $${GOBIN}"


COUNT?=1
TEST?=$(shell go list ./...)
HASHICORP_PACKER_PLUGIN_SDK_VERSION?=$(shell go list -m github.com/hashicorp/packer-plugin-sdk | cut -d " " -f2)
PLUGIN_FQN=$(shell grep -E '^module' <go.mod | sed -E 's/module \s*//')

.PHONY: dev

build:
	@go build -o ${BINARY}

dev:
	go build -ldflags="-X '${PLUGIN_FQN}/version.VersionPrerelease=dev'" -o ${BINARY}
	"$(PACKER)" plugins install --path ${BINARY} "$(shell echo "${PLUGIN_FQN}" | sed 's/packer-plugin-//')"

test:
	@go test -race -count $(COUNT) $(TEST) -timeout=5m
	@go vet ./...

install-packer-sdc: ## Install packer sofware development command
	@go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@${HASHICORP_PACKER_PLUGIN_SDK_VERSION}

plugin-check: build
	@go tool packer-sdc plugin-check ${BINARY}

plugin-smoke: build
	@PACKER="$(PACKER)" bash scripts/test-plugin.sh "$(CURDIR)/${BINARY}"

# testacc: dev
#	@PACKER_ACC=1 go test -count $(COUNT) -v $(TEST) -timeout=120m

generate:
	@go generate ./...
	@rm -rf .docs
	@go tool packer-sdc renderdocs -src docs -partials docs-partials/ -dst .docs/
	@./.web-docs/scripts/compile-to-webdocs.sh "." ".docs" ".web-docs" "ZSvirt"
	@rm -r ".docs"


# Real-cloud tests read a private .env; SCENARIO selects the intended mutation.
SCENARIO?=cloud
.PHONY: testacc test-coverage
testacc:
	python3 scripts/test-acceptance.py "$(SCENARIO)"

test-coverage:
	@mkdir -p .acceptance
	@go test -race -coverprofile=.acceptance/coverage.out ./...
	@go tool cover -func=.acceptance/coverage.out
