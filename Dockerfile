FROM golang:1.11 as builder
ENV CGO_ENABLED=0

RUN mkdir -p /go/src/github.com/openfaas-incubator/vcenter-connector
WORKDIR /go/src/github.com/openfaas-incubator/vcenter-connector

COPY vendor     vendor
COPY pkg        pkg
COPY main.go    .

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"

RUN go test -v ./...

# Stripping via -ldflags "-s -w" 
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o ./connector

FROM alpine:3.10 as ship

RUN addgroup -S app \
    && adduser -S -g app app

WORKDIR /home/app

EXPOSE 8989

COPY --from=builder /go/src/github.com/openfaas-incubator/vcenter-connector/connector    .

RUN chown -R app:app ./

USER app

CMD ["./connector"]