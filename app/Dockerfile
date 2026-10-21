# syntax=docker/dockerfile:1

FROM golang:1.27.1-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/go-chat ./cmd

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/go-chat /go-chat

ENV PORT=8080 GIN_MODE=release
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/go-chat"]
