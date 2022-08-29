#
# skicka Dockerfile
#
# https://github.com/lagerstrom/skicka
#

FROM golang:1.16 as builder
WORKDIR /go/src/app/

COPY cmd ./cmd
COPY static ./static
COPY go.mod .
COPY go.sum .

RUN CGO_ENABLED=0 GOOS=linux go build -o skicka cmd/skicka.go

FROM scratch

COPY --from=0 /go/src/app/skicka .

ENTRYPOINT ["/skicka"]
