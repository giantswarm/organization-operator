# The organization-operator binary is built by architect/go-build and persisted
# to the workspace as `organization-operator-linux-<arch>`.
FROM gcr.io/distroless/static:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931
WORKDIR /
ARG TARGETARCH
COPY organization-operator-linux-${TARGETARCH} /manager
USER 65532:65532
EXPOSE 8080 8000
ENTRYPOINT ["/manager"]
