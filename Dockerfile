FROM golang:1.11 as builder
RUN mkdir -p /go/src/github.com/mendersoftware/deviceadm
WORKDIR /go/src/github.com/mendersoftware/deviceadm
ADD ./ .
RUN CGO_ENABLED=0 GOARCH=amd64 go build -o deviceadm .

FROM alpine:3.4
EXPOSE 8080
RUN mkdir /etc/deviceadm
ENTRYPOINT ["/usr/bin/deviceadm", "--config", "/etc/deviceadm/config.yaml"]
COPY ./config.yaml /etc/deviceadm/
COPY --from=builder /go/src/github.com/mendersoftware/deviceadm/deviceadm /usr/bin/
RUN apk add --update ca-certificates && update-ca-certificates
