GOBIN ?= $$(go env GOPATH)/bin
DOCKER_HOST ?= unix:///Users/guionardo/.orbstack/run/docker.sock

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

gocheck:
	@if ! command -v go >/dev/null 2>&1 ; then \
		echo "\033[31mGO IS NOT INSTALLED\033[0m"; \
		exit 1 ; \
	fi
	@if ! echo "${PATH}" | grep -q "${GOBIN}"; then \
		echo  "\033[31mGO BIN folder is not in PATH: ${GOBIN}\033[0m"; \
		exit 1 ; \
	fi

##@ Dependencies

deps: gocheck install-pre-commit install-golangci install-commitlint install-govulncheck install-go-test-coverage ## Installs/updates dependencies
	@echo "\n🚀 \033[30;44m  ALL DEPENDENCIES ARE INSTALLED  \033[0m"

install-pre-commit:
	@echo  "\n🛠️  \033[30;42m INSTALLING PRE-COMMIT \033[0m"
	@sudo apt install -y pre-commit
	@pre-commit autoupdate
	@pre-commit install -t commit-msg -t pre-commit
	@echo "✅  PRE-COMMIT INSTALLED"

install-golangci:
	@echo  "\n🛠️  \033[30;42m INSTALLING GOLANGCI-LINT \033[0m"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest
	@echo "✅  GOLANGCI-LINT INSTALLED"

install-commitlint:
	@echo  "\n🛠️  \033[30;42m INSTALLING COMMITLINT \033[0m"
	@go install github.com/conventionalcommit/commitlint@latest
	@if ! command -v commitlint >/dev/null 2>&1; then \
		echo "commitlint not found or not accessible yet."; \
		exit 1; \
	fi
	@if [ ! -f .commitlint.yml ] && [ ! -f .commitlint.yaml ] && [ ! -f commitlint.yml ] && [ ! -f commitlint.yaml ]; then \
		echo "\n  No commitlint config file found."; \
		read -p "Do you want to create commitlint config? (y/n): " answer && \
		if [ "$$answer" = "y" ] || [ "$$answer" = "Y" ]; then \
			echo "Creating commitlint config..."; \
			commitlint config create; \
		else \
			echo "Skipping commitlint config creation."; \
		fi; \
	fi
	@echo "✅  COMMITLINT INSTALLED"

install-govulncheck:
	@echo  "\n🛠️  \033[30;42m INSTALLING GOVULNCHECK \033[0m"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "✅  GOVULNCHECK INSTALLED"


##@ Testing

install-go-test-coverage:
	@echo  "\n🛠️  \033[30;42m INSTALLING GO-TEST-COVERAGE \033[0m"
	@go install github.com/vladopajic/go-test-coverage/v2@latest
	@if [ -f .testcoverage.yml ]; then \
		echo "go-test-coverage config file already exists."; \
	else \
		echo "Creating default go-test-coverage config file..."; \
		curl -o .testcoverage.yml -L https://github.com/vladopajic/go-test-coverage/raw/refs/heads/main/.testcoverage.example.yml; \
	fi; \
	@cat .gitignore | grep cover.out >/dev/null  || echo cover.out >> .gitignore
	@echo "✅  GO-TEST-COVERAGE INSTALLED"

check-go-test-coverage:
	@if ! command -v go-test-coverage >/dev/null 2>&1; then \
		echo "\033[31mGO-TEST-COVERAGE IS NOT INSTALLED\033[0m"; \
		echo " Please run 'make deps'"; \
		exit 1; \
	fi

test: ## Run tests
	@go test ./... -v

test-e2e: gocheck ## Run E2E integration tests (requires Docker)
	@echo "\n🚀 \033[30;44m  RUNNING E2E TESTS  \033[0m"
	@DOCKER_HOST=$(DOCKER_HOST) \
		go test ./cache/ -tags=e2e -run 'TestCacheE2E' -v -count=1 -timeout=300s

coverage: check-go-test-coverage check-gocovmerge ## Check test coverage
	@echo "\n🚀 \033[30;44m  RUNNING E2E COVERAGE  \033[0m"
	@DOCKER_HOST=$(DOCKER_HOST) \
		go test -tags=e2e -count=1 -timeout=300s \
		-coverprofile=./cover-e2e.out -covermode=atomic -coverpkg=./... \
		./cache/
	@echo "\n🚀 \033[30;44m  RUNNING UNIT COVERAGE  \033[0m"
	@go test ./... -coverprofile=./cover-unit.out -covermode=atomic -coverpkg=./... -count=1 -timeout=120s
	@echo "\n🚀 \033[30;44m  MERGING COVERAGE PROFILES  \033[0m"
	@gocovmerge ./cover-e2e.out ./cover-unit.out > ./cover.out
	@rm -f ./cover-e2e.out ./cover-unit.out
	@echo "\n🚀 \033[30;44m  CHECKING COVERAGE  \033[0m"
	@go-test-coverage --config=./.testcoverage.yml

check-gocovmerge:
	@if ! command -v gocovmerge >/dev/null 2>&1; then \
		echo "\033[31mGOCOVMERGE IS NOT INSTALLED\033[0m"; \
		echo " Installing..."; \
		go install github.com/wadey/gocovmerge@latest; \
	fi

##@ Linting

check_golangci:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "\033[31mGOLANGCI-LINT IS NOT INSTALLED\033[0m"; \
		echo " Please run 'make deps'"; \
		exit 1; \
	fi

lint: check_golangci ## Run linters
	@golangci-lint run ./...

lint-fix: check_golangci ## Run linters and fix issues
	@golangci-lint run --fix ./...
