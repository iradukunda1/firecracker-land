FROM golang:1.20-alpine as build

WORKDIR /init

COPY init .

RUN go build --tags netgo --ldflags '-s -w -extldflags "-lm -lstdc++ -static"' -o init main.go

FROM alpine:3.15


COPY --from=build /init/init /init

