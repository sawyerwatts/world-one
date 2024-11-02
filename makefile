# Thanks, Alex!
# https://www.alexedwards.net/blog/a-time-saving-makefile-for-your-go-projects

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


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	$(go_build_env) go test -v -race -buildvcs ./...

## test/force: forcefully run all tests
.PHONY: test/force
test/force:
	$(go_build_env) go test -v -race -buildvcs -count=1 ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	$(go_build_env) go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	$(go_build_env) go tool cover -html=/tmp/coverage.out

## audit: run quality control checks
.PHONY: audit
audit: test
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
tidy:
	$(go_build_env) go mod tidy -v
	$(go_build_env) go fmt ./...

## run: run the application via `go run`
.PHONY: run
run:
	$(go_build_env) go run -v -race ${main_package_path}

## run/live: run the application with reloading on file changes
.PHONY: run/live
run/live:
	$(go_build_env) go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "/tmp/bin/${binary_name}" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--misc.clean_on_exit "true"

## build: build the application
.PHONY: build
build:
	# Include additional build steps, like TypeScript, SCSS or Tailwind compilation here...
	GOOS=linux GOARCH=amd64 go build -o=/tmp/bin/${binary_name} ${main_package_path}

## build/run: build the application and run the binary
.PHONY: build/run
build/run: build
	/tmp/bin/${binary_name}

## build/clean: remove build artifacts
.PHONY: build/clean
build/clean:
	rm /tmp/bin/${binary_name}


# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## push: push changes to the remote Git repository
.PHONY: push
push: confirm audit no-dirty
	git push

## production/deploy: deploy the application to production
.PHONY: production/deploy
production/deploy: confirm audit no-dirty
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=/tmp/bin/linux_amd64/${binary_name} ${main_package_path}
	upx -5 /tmp/bin/linux_amd64/${binary_name}
	# TODO: Include additional deployment steps here...

