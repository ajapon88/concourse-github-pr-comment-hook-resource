FROM golang:1.13 as builder

COPY . /src
WORKDIR /src

ENV CGO_ENABLED 0

RUN go get -d ./... \
    && go build -o /assets/in ./cmd/in \
    && go build -o /assets/out ./cmd/out \
    && go build -o /assets/check ./cmd/check


FROM alpine:edge AS resource

RUN apk add --no-cache bash
COPY --from=builder assets/ /opt/resource/
RUN chmod +x /opt/resource/*


FROM resource
