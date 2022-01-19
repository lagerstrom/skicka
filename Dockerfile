#
# skicka Dockerfile
#
# https://github.com/lagerstrom/skicka
#

FROM golang:1.16 as builder
WORKDIR /go/src/app/

COPY src ./src
COPY html ./html
COPY go.mod .
COPY go.sum .

RUN go install github.com/gobuffalo/packr/v2/packr2@v2.8.3
RUN cd src; CGO_ENABLED=0 GOOS=linux packr2 build -o ../skicka main.go

FROM scratch

COPY --from=0 /go/src/app/skicka .

ENTRYPOINT ["/skicka"]
