ARG GO_VERSION
FROM golang:${GO_VERSION} as build

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build ./cmd/exporter

FROM gcr.io/distroless/static

WORKDIR /app
COPY --from=build /build/exporter exporter

ENTRYPOINT ["/app/exporter"]
