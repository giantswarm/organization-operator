# The organization-operator binary is built by architect/go-build and persisted
# to the workspace as `organization-operator-linux-<arch>`.
FROM gcr.io/distroless/static:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5
WORKDIR /
ARG TARGETARCH
COPY organization-operator-linux-${TARGETARCH} /manager
USER 65532:65532
EXPOSE 8080 8000
ENTRYPOINT ["/manager"]
