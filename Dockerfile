FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY exporter-filterproxy proxy
USER 65532:65532

CMD ["/proxy"]
