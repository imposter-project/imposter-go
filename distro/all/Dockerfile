FROM golang:1.23 AS build
LABEL authors="Imposter Project"

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

ADD . /go/src/github.com/imposter-project/imposter-go
WORKDIR /go/src/github.com/imposter-project/imposter-go
RUN go get -d -v ./...
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

# Build the main binary
RUN go build -tags lambda.norpc -ldflags "-s -w -X github.com/imposter-project/imposter-go/internal/version.Version=${VERSION}" -o /imposter-go ./cmd/imposter

# Build plugins for the target platform
RUN mkdir -p /opt/imposter/plugins
RUN for p in $(cd ./external/plugins && ls); do \
        echo "Building plugin $p for ${TARGETOS}/${TARGETARCH}"; \
        if [ "${TARGETOS}" = "windows" ]; then \
            go build -tags lambda.norpc -ldflags "-X main.Version=${VERSION}" -o /opt/imposter/plugins/plugin-$p.exe ./external/plugins/$p; \
        else \
            go build -tags lambda.norpc -ldflags "-X main.Version=${VERSION}" -o /opt/imposter/plugins/plugin-$p ./external/plugins/$p; \
        fi; \
    done

# Create required directories
RUN mkdir -p /tmp /opt/imposter/config

FROM scratch
COPY --from=build /imposter-go /imposter-go
COPY --from=build --chmod=0600 /tmp /tmp/
COPY --from=build --chmod=0755 /opt/imposter/config /opt/imposter/config/
COPY --from=build --chmod=0755 /opt/imposter/plugins /opt/imposter/plugins/

ENV IMPOSTER_CONFIG_DIR=/opt/imposter/config
ENV IMPOSTER_PLUGIN_DIR=/opt/imposter/plugins
ENV IMPOSTER_EXTERNAL_PLUGINS=true
ENTRYPOINT ["/imposter-go"]