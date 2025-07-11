VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/imposter-project/imposter-go/internal/version.Version=$(VERSION)

.PHONY: build
build:
	go build -tags lambda.norpc -ldflags "$(LDFLAGS)" -o imposter-go ./cmd/imposter

.PHONY: build-prod
build-prod:
	go build -tags lambda.norpc -ldflags "$(LDFLAGS) -s -w" -trimpath -o imposter-go ./cmd/imposter

.PHONY: build-plugins
build-plugins:
	mkdir -p bin
	for p in $$( cd ./external/plugins && ls ); do \
		echo "Building plugin $$p"; \
		if [ "$(shell go env GOOS)" = "windows" ]; then \
			go build -tags lambda.norpc -ldflags "-X main.Version=$(VERSION)" -o ./bin/plugin-$$p.exe ./external/plugins/$$p; \
		else \
			go build -tags lambda.norpc -ldflags "-X main.Version=$(VERSION)" -o ./bin/plugin-$$p ./external/plugins/$$p; \
		fi; \
	done

.PHONY: fmt
fmt:
	go fmt ./... 

.PHONY: install
install:
	go install -tags lambda.norpc -ldflags "$(LDFLAGS)" ./cmd/imposter

.PHONY: run
run:
	go run -tags lambda.norpc -ldflags "$(LDFLAGS)" ./cmd/imposter/main.go $(filter-out $@,$(MAKECMDGOALS))

.PHONY: test
test:
	go test ./... 

.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: coverage-html
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html 
