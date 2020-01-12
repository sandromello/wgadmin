FROM gcr.io/distroless/base:nonroot
COPY wgadmin /usr/bin/wgadmin
ENTRYPOINT ["/usr/bin/wgadmin"]
