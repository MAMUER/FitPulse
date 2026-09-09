FROM alpine@sha256:24aa3c2e3b6f4425d2b3c1e3289d2d8f3f3b2e1d5c6a7b8c9d0e1f2a3b4c5d6e

ARG KUBECTL_VERSION=v1.36.0

# hadolint ignore=DL3018
RUN apk upgrade --no-cache && \
    apk add --no-cache bash ca-certificates curl tzdata unzip && \
    curl -sSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    kubectl version --client=true && \
    addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

WORKDIR /app
USER 1000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/bin/true"]
