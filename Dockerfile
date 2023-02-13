FROM alpine:3.17.2

RUN apk add --no-cache ca-certificates

ADD ./organization-operator /organization-operator

ENTRYPOINT ["/organization-operator"]
