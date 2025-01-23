FROM golang:1.23 AS build
LABEL authors="Imposter Project"

ARG VERSION=dev

ADD . /go/src/github.com/imposter-go/imposter-go
WORKDIR /go/src/github.com/imposter-go/imposter-go
RUN go get -d -v ./...
ENV CGO_ENABLED=0
RUN go build -tags lambda.norpc -ldflags "-s -w -X github.com/imposter-project/imposter-go/internal/version.Version=${VERSION}" -o /imposter-go ./cmd/imposter

FROM scratch
COPY --from=build /imposter-go /imposter-go
ENV IMPOSTER_CONFIG_DIR=/opt/imposter/config
ENTRYPOINT ["/imposter-go"]
