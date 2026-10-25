# Packs the static Linux binaries in bin/ (build context), one per platform: built by the CI job binary,
# or locally by make e2e. Building from source is docker/Dockerfile.
FROM gcr.io/distroless/static:nonroot
ARG TARGETARCH
LABEL org.opencontainers.image.source=https://github.com/radiopath/radiopath
COPY radiopath-linux-${TARGETARCH} /radiopath
ENV RADIOPATH_LISTEN=:8080 RADIOPATH_DEM_DIR=/data/dem
EXPOSE 8080 9090
USER nonroot
ENTRYPOINT ["/radiopath"]
