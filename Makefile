VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/imposter-project/imposter-go/internal/version.Version=$(VERSION)

.PHONY: build
build:
	go build -tags lambda.norpc -ldflags "$(LDFLAGS)" -o imposter-go ./cmd/imposter

.PHONY: install
install:
	go install -tags lambda.norpc -ldflags "$(LDFLAGS)" -o "${GOPATH}/bin/imposter-go" ./cmd/imposter

.PHONY: run
run:
	go run -tags lambda.norpc -ldflags "$(LDFLAGS)" ./cmd/imposter/main.go $(filter-out $@,$(MAKECMDGOALS))

.PHONY: test
test:
	go test -v ./... 

.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: coverage-html
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html 