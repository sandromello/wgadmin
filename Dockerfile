FROM busybox AS build-env

WORKDIR /build
COPY dist/wgadmin_Linux_x86_64.tar.gz /tmp/wgadmin_Linux_x86_64.tar.gz
RUN tar -xf /tmp/wgadmin_Linux_x86_64.tar.gz -C /build/ wgadmin && rm -f /tmp/wgadmin_Linux_x86_64.tar.gz

FROM gcr.io/distroless/base:nonroot
COPY --from=build-env /build/wgadmin /usr/bin/wgadmin
ENTRYPOINT ["/usr/bin/wgadmin"]
