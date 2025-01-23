FROM golang:1.23 AS build
LABEL authors="Imposter Project"

ARG VERSION=dev

ADD . /go/src/github.com/imposter-go/imposter-go
WORKDIR /go/src/github.com/imposter-go/imposter-go
RUN go get -d -v ./...
ENV CGO_ENABLED=0
RUN go build -tags lambda.norpc -ldflags "-s -w -X github.com/imposter-project/imposter-go/internal/version.Version=${VERSION}" -o /imposter-go ./cmd/imposter
RUN mkdir -p /tmp /opt/imposter/config

FROM scratch
COPY --from=build /imposter-go /imposter-go
COPY --from=build --chmod=0600 /tmp /tmp/
COPY --from=build --chmod=0400 /opt/imposter/config /opt/imposter/config/

ENV IMPOSTER_CONFIG_DIR=/opt/imposter/config
ENTRYPOINT ["/imposter-go"]
