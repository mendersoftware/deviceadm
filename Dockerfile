FROM iron/base

COPY ./deviceadm /usr/bin/

RUN mkdir /etc/deviceadm
COPY ./config.yaml /etc/deviceadm/

ENTRYPOINT ["/usr/bin/deviceadm", "-config", "/etc/deviceadm/config.yaml"]
