FROM golang:1.10.4 as builder

RUN go get -u github.com/vmware/govmomi/vcsim

FROM vmware/photon:2.0

COPY --from=builder /go/bin/vcsim .
CMD ["./vcsim", "-httptest.serve", "0.0.0.0:443"]