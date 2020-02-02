FROM alpine:3.11

RUN apk add ca-certificates && rm -rf /var/cache/apk/*
RUN addgroup -g 1001 -S wgadm && adduser -u 1001 -S -G wgadm wgadm
WORKDIR /home/wgadm

COPY wgadmin /usr/bin/wgadmin
COPY --chown=wgadm:wgadm web/ web/
USER wgadm

ENTRYPOINT ["/usr/bin/wgadmin"]
