# Build terraform-provider-openfga from source with the patches of hack/upstream applied: the
# released binary does not carry them, see hack/upstream/README.md. Built on the build platform,
# cross-compiled for the target.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS tfprovider
RUN apk --no-cache add git
ARG TARGETOS
ARG TARGETARCH
ARG TERRAFORM_PROVIDER_REPO
ARG TERRAFORM_PROVIDER_VERSION
ARG TERRAFORM_NATIVE_PROVIDER_BINARY
RUN git clone --depth 1 --branch "v${TERRAFORM_PROVIDER_VERSION}" "${TERRAFORM_PROVIDER_REPO}" /src
WORKDIR /src
COPY upstream/ /patches/
RUN for p in /patches/*.patch; do git apply --verbose "$p" || exit 1; done
# The released terraform-provider-openfga pins dependencies with known vulnerabilities in its go.mod
# (govulncheck on the binary). Raised here until upstream does, and tracked by Renovate through the
# markers: drop an ARG once the pinned upstream version already requires at least that version.
# renovate: datasource=go depName=golang.org/x/net
ARG GO_X_NET_VERSION=v0.60.0
# renovate: datasource=go depName=golang.org/x/text
ARG GO_X_TEXT_VERSION=v0.42.0
# renovate: datasource=go depName=google.golang.org/grpc
ARG GRPC_VERSION=v1.83.2
# renovate: datasource=go depName=go.opentelemetry.io/otel
ARG OTEL_VERSION=v1.45.0
RUN go get "golang.org/x/net@${GO_X_NET_VERSION}" "golang.org/x/text@${GO_X_TEXT_VERSION}" \
    "google.golang.org/grpc@${GRPC_VERSION}" "go.opentelemetry.io/otel@${OTEL_VERSION}" && go mod tidy
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${TERRAFORM_PROVIDER_VERSION}" \
    -o "/out/${TERRAFORM_NATIVE_PROVIDER_BINARY}" .

FROM alpine:3.23.4
RUN apk --no-cache add ca-certificates bash

ARG TARGETOS
ARG TARGETARCH

ADD "bin/${TARGETOS}_${TARGETARCH}/provider" /usr/local/bin/provider

ENV USER_ID=65532

# Setup Terraform environment

## Provider-dependent configuration
ARG TERRAFORM_VERSION
ARG TERRAFORM_PROVIDER_SOURCE
ARG TERRAFORM_PROVIDER_VERSION
ARG TERRAFORM_NATIVE_PROVIDER_BINARY

## End of - Provider-dependent configuration

ENV PLUGIN_DIR=/terraform/provider-mirror/registry.terraform.io/${TERRAFORM_PROVIDER_SOURCE}/${TERRAFORM_PROVIDER_VERSION}/${TARGETOS}_${TARGETARCH}
ENV TF_CLI_CONFIG_FILE=/terraform/.terraformrc
ENV TF_FORK=0

RUN mkdir -p ${PLUGIN_DIR}

ADD https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${TARGETOS}_${TARGETARCH}.zip /tmp
ADD terraformrc.hcl ${TF_CLI_CONFIG_FILE}

RUN unzip /tmp/terraform_${TERRAFORM_VERSION}_${TARGETOS}_${TARGETARCH}.zip -d /usr/local/bin \
  && chmod +x /usr/local/bin/terraform \
  && rm /tmp/terraform_${TERRAFORM_VERSION}_${TARGETOS}_${TARGETARCH}.zip \
  && chown -R ${USER_ID}:${USER_ID} /terraform

COPY --from=tfprovider --chown=${USER_ID}:${USER_ID} --chmod=0755 \
  /out/${TERRAFORM_NATIVE_PROVIDER_BINARY} ${PLUGIN_DIR}/${TERRAFORM_NATIVE_PROVIDER_BINARY}
# End of - Setup Terraform environment

# Provider controller needs these environment variable at runtime
ENV TERRAFORM_VERSION=${TERRAFORM_VERSION}
ENV TERRAFORM_PROVIDER_SOURCE=${TERRAFORM_PROVIDER_SOURCE}
ENV TERRAFORM_PROVIDER_VERSION=${TERRAFORM_PROVIDER_VERSION}
ENV TERRAFORM_NATIVE_PROVIDER_PATH=${PLUGIN_DIR}/${TERRAFORM_NATIVE_PROVIDER_BINARY}

USER ${USER_ID}
EXPOSE 8080

ENTRYPOINT ["provider"]
