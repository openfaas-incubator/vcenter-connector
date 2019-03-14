FROM golang:1.10.4 as builder
RUN mkdir -p /go/src/github.com/openfaas-incubator/vcenter-connector
WORKDIR /go/src/github.com/openfaas-incubator/vcenter-connector

COPY vendor     vendor
COPY pkg        pkg
COPY main.go    .

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"

RUN go test -v ./...

# Stripping via -ldflags "-s -w" 
RUN GOARM=7 CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o ./connector

FROM alpine:3.8

EXPOSE 8989

COPY --from=builder /go/src/github.com/openfaas-incubator/vcenter-connector/    .

CMD ["./connector"]