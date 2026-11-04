# syntax=docker/dockerfile:1

# Build on the host's own architecture and cross-compile to the target, so a
# multi-arch build never pays for QEMU emulation. Both default to the build
# host when no platform is requested, so a plain `docker build` is unchanged.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS build
WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/go-chat ./cmd

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/go-chat /go-chat

ENV PORT=8080 GIN_MODE=release
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/go-chat"]
