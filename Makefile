SHELL := /bin/bash
VERSION := v0.0.3
NAME := negotiator

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
	go test -v --cover -cpu=2 `go list ./... | grep -v /vendor/`

.PHONY: test-all
test-all:
	go test -cpu=2 -cover `go list ./... | grep -v /vendor/` -integration=true	

.PHONY: test-race
test-race:
    go test -v -cpu=1,2,4 -short -race `go list ./... | grep -v /vendor/`

.PHONY: test-with-vendor 
test-with-vendor:
	go test -v -cpu=2 ./...

build:
	cd cmd/negotiator && go build -ldflags "-X main.Version=$(VERSION)"
	cd cmd/services && go build -ldflags "-X main.Version=$(VERSION)"

build_linux:
	cd cmd/negotiator && env GOOS=linux go build -ldflags "-X main.Version=$(VERSION)"
	cd cmd/services && env GOOS=linux go build -ldflags "-X main.Version=$(VERSION)"

deps:
	go get github.com/c4milo/github-release
	go get github.com/mitchellh/gox
	go get -u github.com/goadesign/goa/...

compile:
	@rm -rf build/
	@gox -ldflags "-X main.Version=$(VERSION)" \
	-osarch="darwin/amd64" \
	-osarch="linux/amd64" \
	-output "build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)" \
	./...

dist: compile
	$(eval FILES := $(shell ls build))
	@rm -rf dist && mkdir dist
	@for f in $(FILES); do \
		(cd $(shell pwd)/build/$$f && tar -cvzf ../../dist/$$f.tar.gz *); \
		(cd $(shell pwd)/dist && shasum -a 512 $$f.tar.gz > $$f.sha512); \
		echo $$f; \
	done

release: dist
	@latest_tag=$$(git describe --tags `git rev-list --tags --max-count=1`); \
	comparison="$$latest_tag..HEAD"; \
	if [ -z "$$latest_tag" ]; then comparison=""; fi; \
	changelog=$$(git log $$comparison --oneline --no-merges --reverse); \
	github-release feedhenry/$(NAME) $(VERSION) "$$(git rev-parse --abbrev-ref HEAD)" "**Changelog**<br/>$$changelog" 'dist/*';
