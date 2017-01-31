FROM alpine:3.4

EXPOSE 8080

COPY ./deviceadm /usr/bin/

RUN mkdir /etc/deviceadm
COPY ./config.yaml /etc/deviceadm/

ENTRYPOINT ["/usr/bin/deviceadm", "-config", "/etc/deviceadm/config.yaml"]
