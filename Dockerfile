# The organization-operator binary is built by architect/go-build and persisted
# to the workspace as `organization-operator-linux-<arch>`.
FROM gcr.io/distroless/static:nonroot
WORKDIR /
ARG TARGETARCH
COPY organization-operator-linux-${TARGETARCH} /manager
USER 65532:65532
EXPOSE 8080 8000
ENTRYPOINT ["/manager"]
