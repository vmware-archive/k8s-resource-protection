FROM golang:1.12
WORKDIR /go/src/github.com/k14s/pv-labeling-controller/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o controller ./cmd/controller/...
RUN mkdir -p /tmp

FROM scratch
COPY --from=0 /go/src/github.com/k14s/pv-labeling-controller/controller .
COPY --from=0 /tmp /tmp
ENTRYPOINT ["/controller"]
