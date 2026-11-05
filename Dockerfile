ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -D -H -u 10001 -G app app

WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/seed /app/seed

USER app
EXPOSE 3333
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1:3333/health || exit 1
CMD ["/app/server"]
