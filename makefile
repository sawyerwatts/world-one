# Thanks, Alex!
# https://www.alexedwards.net/blog/a-time-saving-makefile-for-your-go-projects

SHELL := /bin/bash

.DEFAULT_GOAL = run

main_package_path = ./cmd/world-one
binary_name = world-one

# NOTE: Don't forget about build tags, like `-tags integration`!

# WARN: It can be helpful to set GOPROXY=direct when running `audit` as that
# will use the cached versions.

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

# NOTE: Starting a line with @ (no subsequent space) will disable printing the
# line that is being executed.

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	@test -z "$(shell git status --porcelain)"

.PHONY: source-env
source-env:
	source ./env


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## test: run all tests
.PHONY: test
test: source-env
	$(go_build_env) go test -v -race -buildvcs ./...

## test/force: forcefully run all tests
.PHONY: test/force
test/force: source-env
	$(go_build_env) go test -v -race -buildvcs -count=1 ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover: source-env
	$(go_build_env) go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	$(go_build_env) go tool cover -html=/tmp/coverage.out

## audit: run quality control checks
.PHONY: audit
audit: source-env test tools/sqlc/vet
	$(go_build_env) go mod tidy -diff
	$(go_build_env) go mod verify
	test -z "$(shell gofmt -l .)"
	$(go_build_env) go vet ./...
	# WARN: You may decide to freeze these versions within a project
	$(go_build_env) go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	$(go_build_env) go run golang.org/x/vuln/cmd/govulncheck@latest ./...


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy: source-env
	$(go_build_env) go mod tidy -v
	$(go_build_env) go fmt ./...

## run: run the application via `go run`
.PHONY: run
run: source-env
	$(go_build_env) go run -v -race ${main_package_path}

## run/live: run the application with reloading on file changes
.PHONY: run/live
run/live: source-env
	$(go_build_env) go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "/tmp/bin/${binary_name}" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--misc.clean_on_exit "true"

## build: build the application
.PHONY: build
build: source-env
	# Include additional build steps, like TypeScript, SCSS or Tailwind compilation here...
	GOOS=linux GOARCH=amd64 go build -o=/tmp/bin/${binary_name} ${main_package_path}

## build/run: build the application and run the binary
.PHONY: build/run
build/run: source-env build
	/tmp/bin/${binary_name}

## build/clean: remove build artifacts
.PHONY: build/clean
build/clean: source-env
	rm /tmp/bin/${binary_name}

## tools/sqlc/vet: vet/lint the sqlc queries
.PHONY: tools/sqlc/vet
tools/sqlc/vet: source-env
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest vet

## tools/sqlc/generate: generate code from sqlc queries
.PHONY: tools/sqlc/generate
tools/sqlc/generate: source-env
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

## stub-.env: create a stubbed .env file
.PHONY: stub-.env
stub-.env:
	@echo -e "#!/bin/bash \n\
	# This script contains credentials and secrets, so it is present in the .gitignore \n\
	export W1_PGHOST=TODO:this \n\
	export W1_PGUSER=TODO:this \n\
	export W1_PGPASSWORD=TODO:this" > .env
	@echo ".env has been stubbed, now go initialize the values"


# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## push: push changes to the remote Git repository
.PHONY: push
push: source-env confirm audit no-dirty
	git push

## production/deploy: deploy the application to production
.PHONY: production/deploy
production/deploy: source-env confirm audit no-dirty source-env
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=/tmp/bin/linux_amd64/${binary_name} ${main_package_path}
	upx -5 /tmp/bin/linux_amd64/${binary_name}
	# TODO: Include additional deployment steps here...

