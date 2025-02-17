FROM docker.io/golang:1.24.0-alpine3.21 AS builder

# add /etc/mime.types
RUN apk add mailcap  

WORKDIR /go/src/app

RUN mkdir -p /build/tmp

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/crydrv




FROM scratch

EXPOSE 8000/tcp

WORKDIR /

VOLUME [ "/www" ]

USER 1000:1000

COPY --from=builder --chmod=0500 --chown=1000:1000 /build/crydrv /crydrv
COPY --from=builder --chmod=0700 --chown=1000:1000 /build/tmp /tmp
COPY --from=builder --chmod=0444 --chown=0:0 /etc/mime.types /etc/mime.types

CMD ["/crydrv"]
