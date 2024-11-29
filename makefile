# Kudos to Alex for the original version:
# https://www.alexedwards.net/blog/a-time-saving-makefile-for-your-go-projects

SHELL := /bin/bash

.DEFAULT_GOAL = run

main_package_path = ./cmd/world-one
binary_name = world-one
go_env = GOOS=linux GOARCH=amd64

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

.PHONY: source-.env
source-.env:
	@source ./.env || (echo 'Run `make stub-.env and fill out the environment variables'; exit 1)


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## test: run all tests
.PHONY: test
test: source-.env
	$(go_env) go test -v -race -buildvcs ./...

## test/force: forcefully run all tests
.PHONY: test/force
test/force: source-.env
	$(go_env) go test -v -race -buildvcs -count=1 ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover: source-.env
	$(go_env) go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	$(go_env) go tool cover -html=/tmp/coverage.out

## audit: run quality control checks
.PHONY: audit
audit: source-.env test tools/sqlc/vet
	$(go_env) go mod tidy -diff
	$(go_env) go mod verify
	test -z "$(shell gofmt -l .)"
	$(go_env) go vet ./...
	# WARN: You may decide to freeze these versions within a project
	$(go_env) go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	$(go_env) go run golang.org/x/vuln/cmd/govulncheck@latest ./...


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy: source-.env
	$(go_env) go mod tidy -v
	$(go_env) go fmt ./...

## run: build the application as debug and run the binary
.PHONY: run
run: source-.env build/debug
	/tmp/bin/${binary_name}

## run/live: run the application with reloading on file changes
.PHONY: run/live
run/live: source-.env build/debug
	$(go_env) go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "/tmp/bin/${binary_name}" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--misc.clean_on_exit "true"

## build: build the application without -race, -v, etc
.PHONY: build
build: source-.env
	$(go_env) go build -o=/tmp/bin/${binary_name} ${main_package_path}

## build/race: build the application with -race, -v, etc
.PHONY: build/debug
build/debug: source-.env
	$(go_env) go build -v -race -o=/tmp/bin/${binary_name} ${main_package_path}

## build/clean: remove build artifacts
.PHONY: build/clean
build/clean: source-.env
	rm /tmp/bin/${binary_name}

## tools/sqlc/vet: vet/lint the sqlc queries
.PHONY: tools/sqlc/vet
tools/sqlc/vet: source-.env
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest vet

## tools/sqlc/generate: generate code from sqlc queries
.PHONY: tools/sqlc/generate
tools/sqlc/generate: source-.env
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

## stub-.env: create a stubbed .env file
.PHONY: stub-.env
stub-.env:
	@echo -e "#!/bin/bash \n\
	# This script contains credentials and secrets, so it is present in the .gitignore \n\
	export W1_PGURL="postgresql://YOUR_USER_NAME:YOUR_PASSWORD@localhost/world_one?sslmode=disable" \n\
	alias migrate='go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest' \n\
	# $ migrate create -ext sql -dir sql/migrations -seq create_users_table \n\
	# $ source ./env.sh; migrate -path sql/migrations/ -database $W1_PGURL up \n\
	# If you need to force a certain version: \n\
	# 	$ migrate -path sql/migrations/ -database $W1_PGURL force 1 \n\
	" > .env
	@echo ".env has been stubbed, please go initialize the values"


# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## push: push changes to the remote Git repository
.PHONY: push
push: confirm source-.env audit no-dirty
	git push

## production/deploy: deploy the application to production
.PHONY: production/deploy
production/deploy: source-.env confirm audit no-dirty source-.env
	$(go_env) go build -ldflags='-s' -o=/tmp/bin/linux_amd64/${binary_name} ${main_package_path}
	upx -5 /tmp/bin/linux_amd64/${binary_name}
	# TODO: Include additional deployment steps here...

