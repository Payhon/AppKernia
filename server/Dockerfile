FROM golang:1.26.5-alpine AS build

RUN apk add --no-cache ca-certificates git
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ak-api ./cmd/ak-api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ak-cli ./cmd/ak-cli \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ak-worker ./cmd/ak-worker

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appkernia \
    && adduser -S -G appkernia -h /app appkernia \
    && mkdir -p /app/data/object-storage \
    && chown -R appkernia:appkernia /app/data
WORKDIR /app/server
COPY --from=build /out/ /app/bin/
COPY blueprint/ /app/blueprint/
USER appkernia
EXPOSE 8080
ENTRYPOINT ["/app/bin/ak-api"]
