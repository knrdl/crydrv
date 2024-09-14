FROM docker.io/golang:1.23.1-alpine3.20 AS builder

WORKDIR /go/src/app

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /crydrv




FROM scratch

EXPOSE 8000/tcp

WORKDIR /

VOLUME [ "/www" ]

USER 1000:1000

COPY --from=builder --chmod=0500 --chown=1000:1000 /crydrv /crydrv

CMD ["/crydrv"]