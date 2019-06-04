FROM golang:1.11.4-alpine as builder

RUN apk add git --no-cache

WORKDIR /root/go/src/github.com/vmware/
RUN git clone https://github.com/vmware/govmomi

WORKDIR /root/go/src/github.com/vmware/govmomi
ENV GOPATH=/root/go/
# Building using -mod=vendor, which will utilize the v
RUN CGO_ENABLED=0 GOOS=linux go build -o vcsim/vcsim vcsim/main.go

FROM alpine:3.9

WORKDIR /root/

EXPOSE 8989
COPY --from=builder /root/go/src/github.com/vmware/govmomi/vcsim/vcsim .

CMD ["./vcsim", "-httptest.serve", ":8989", "-tls=false"]
