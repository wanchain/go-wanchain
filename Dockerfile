# Build Gwan in a stock Go builder container
FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc git musl-dev linux-headers

ADD . /go-wanchain
RUN cd /go-wanchain && make gwan

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-wanchain/build/bin/gwan /usr/local/bin/

EXPOSE 8545 17717/tcp 17717/udp
ENTRYPOINT ["gwan"]
