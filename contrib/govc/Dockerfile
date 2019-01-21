FROM golang:1.11.4-alpine as builder

RUN apk add git --no-cache

WORKDIR /root/go/src/github.com/vmware/
RUN git clone https://github.com/vmware/govmomi

WORKDIR /root/go/src/github.com/vmware/govmomi
ENV GOPATH=/root/go/
RUN CGO_ENABLED=0 GOOS=linux go build -o govc/govc govc/main.go

FROM alpine:3.8

WORKDIR /root/

EXPOSE 8989
COPY --from=builder //root/go/src/github.com/vmware/govmomi/govc/govc .

CMD ["./govc"]
