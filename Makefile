SHELL := /bin/bash
VERSION := 0.0.7
NAME := negotiator

# To build for a os you are not on, use (for_(linux|mac|windows) must be called first):
# make for_linux build

# To compile for linux, create the docker image 
# and push it to docker.io/feedhenry/negotiator with the version specified above, use: 
# make docker_build_push

# This is the first target, so it is the default, i.e.
# make for_linux is the same as make for_linux build
.PHONY: build
build: build_negotiator build_jobs build_services

.PHONY: for_linux
for_linux:
	export GOOS=linux
.PHONY: for_mac
for_mac:
	export GOOS=darwin

.PHONY: for_windows
for_windows:
	export GOOS=windows

.PHONY: phils-test
phils-test:
	echo $(MAKECMDGOALS)

.PHONY: all
all:
	@go install -v

.PHONY: clean
clean:
	@-go clean -i

.PHONY: ci
ci: test test-race

# goimports doesn't support the -s flag to simplify code, therefore we use both
# goimports and gofmt -s.
.PHONY: check-gofmt
check-gofmt:
	diff <(gofmt -d -s .) <(printf "")

.PHONY: check-golint
check-golint:
	diff <(golint ./... | grep -v vendor/) <(printf "")

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test-unit:
	go test -v --cover -cpu=2 `go list ./... | grep -v /vendor/ | grep -v /design`

.PHONY: test-race
test-race:
    go test -v -cpu=1,2,4 -short -race `go list ./... | grep -v /vendor/`

.PHONY: build_negotiator
build_negotiator:
	cd cmd/negotiator && go build -ldflags "-X main.Version=v$(VERSION)"

.PHONY: build_jobs
build_jobs:
	cd cmd/jobs && go build -ldflags "-X main.Version=v$(VERSION)"

.PHONY: build_services
build_services:
	cd cmd/services && go build -ldflags "-X main.Version=v$(VERSION)"

.PHONY: docker_build
docker_build:
	docker build -t feedhenry/negotiator:${VERSION} .

.PHONY: docker_push
docker_push:
	docker push feedhenry/negotiator:${VERSION}

.PHONY: docker_build_push
docker_build_push: for_linux build docker_build docker_push

.PHONY: deps
deps:
	go get github.com/c4milo/github-release
	go get github.com/mitchellh/gox
	go get -u github.com/goadesign/goa/...

.PHONY: compile
compile:
	@rm -rf build/
	@gox -ldflags "-X main.Version=v$(VERSION)" \
	-osarch="darwin/amd64" \
	-osarch="linux/amd64" \
	-output "build/{{.Dir}}_v$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)" \
	./...

dist: compile
	$(eval FILES := $(shell ls build))
	@rm -rf dist && mkdir dist
	@for f in $(FILES); do \
		(cd $(shell pwd)/build/$$f && tar -cvzf ../../dist/$$f.tar.gz *); \
		(cd $(shell pwd)/dist && shasum -a 512 $$f.tar.gz > $$f.sha512); \
		echo $$f; \
	done

.PHONY: release
release: dist
	@latest_tag=$$(git describe --tags `git rev-list --tags --max-count=1`); \
	comparison="$$latest_tag..HEAD"; \
	if [ -z "$$latest_tag" ]; then comparison=""; fi; \
	changelog=$$(git log $$comparison --oneline --no-merges --reverse); \
	github-release feedhenry/$(NAME) v$(VERSION) "$$(git rev-parse --abbrev-ref HEAD)" "**Changelog**<br/>$$changelog" 'dist/*';
