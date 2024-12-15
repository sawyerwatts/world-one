# Kudos to Alex for the original version:
# https://www.alexedwards.net/blog/a-time-saving-makefile-for-your-go-projects

main_package_path = ./cmd/world-one
binary_name = world-one

# NOTE: Don't forget about build tags (which can be negated).

# WARN: It can be helpful to set GOPROXY=direct when running `audit` as that
# will use the cached versions.

# NOTE: Starting a line with @ (no subsequent space) will disable printing the
# line that is being executed.

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

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


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	go test -v -race -shuffle=on -parallel=8 -buildvcs ./...

# TODO: nail down testing strategy and consider more testing recipes

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -shuffle=on -parallel=8 -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## audit: run quality control checks
.PHONY: audit
audit: test tools/sqlc/vet
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)"
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@v0.5.1 -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.3 ./...


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## stub-.env: create a stubbed .env file
.PHONY: stub-.env
stub-.env:
	@echo -e "#!/bin/bash \n\
	set -u \n\
	# This script contains credentials and secrets, so it is present in the .gitignore \n\
	export W1_PGURL="postgresql://YOUR_USER_NAME:YOUR_PASSWORD@localhost/world_one?sslmode=disable" \n\
	alias migrate='go run -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1' \n\
	# $ migrate create -ext sql -dir sql/migrations -seq create_users_table \n\
	# $ source ./env.sh; migrate -path sql/migrations/ -database $W1_PGURL up \n\
	# If you need to force a certain version: \n\
	# 	$ migrate -path sql/migrations/ -database $W1_PGURL force 1 \n\
	" > .env
	@echo ".env has been stubbed, please go initialize the values"


## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fmt ./...

## run: build/local and run the binary using .env
.PHONY: run
run: build/debug
	source ./.env && /tmp/bin/${binary_name}

## build/local: build the application with -race, -v, etc
.PHONY: build/local
build/local:
	go build -v -race -o=/tmp/bin/${binary_name} ${main_package_path}

## build/release: build the application without -race, -v, etc, for a specific OS and architecture
.PHONY: build/release
build/release:
	GOOS=linux GOARCH=amd64 go build -o=/tmp/bin/${binary_name} ${main_package_path}

## build/clean: remove build artifacts
.PHONY: build/clean
build/clean:
	rm /tmp/bin/${binary_name}

## tools/sqlc/vet: vet/lint the sqlc queries
.PHONY: tools/sqlc/vet
tools/sqlc/vet:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 vet

## tools/sqlc/generate: generate code from sqlc queries
.PHONY: tools/sqlc/generate
tools/sqlc/generate:
	source ./.env && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate


# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## push: push changes to the remote Git repository
.PHONY: push
push: confirm audit no-dirty
	git push

## production/deploy: deploy the application to production
.PHONY: production/deploy
production/deploy: confirm audit no-dirty build/release
	upx -5 /tmp/bin/${binary_name}
	# TODO: need to tar website/ and deploy too
	# TODO: replace the host in web api specs w/ real value
	# TODO: Include additional deployment steps here...

