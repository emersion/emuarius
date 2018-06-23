FROM golang:1.9.7-alpine3.7 AS builder

RUN apk add --no-cache curl git
RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 && chmod +x /usr/local/bin/dep

RUN mkdir -p /go/src/github.com/emersion/emuarius
WORKDIR /go/src/github.com/emersion/emuarius

ADD . ./

RUN dep ensure -vendor-only
RUN ls vendor/github.com/emersion
RUN go build -o emuarius cmd/emuarius/main.go
RUN ls -la

FROM alpine:3.7
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /go/src/github.com/emersion/emuarius/emuarius /app/emuarius
RUN chmod +x emuarius
CMD ./emuarius
