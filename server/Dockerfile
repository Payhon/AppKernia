FROM golang:1.26.5-alpine AS build

ARG AK_APP_VERSION=dev
ARG AK_GIT_COMMIT=unknown
ARG AK_BUILD_TIME=unknown

RUN apk add --no-cache ca-certificates git
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN BUILD_LDFLAGS="-s -w -X github.com/appkernia/appkernia/server/internal/platform/buildinfo.Version=${AK_APP_VERSION} -X github.com/appkernia/appkernia/server/internal/platform/buildinfo.Commit=${AK_GIT_COMMIT} -X github.com/appkernia/appkernia/server/internal/platform/buildinfo.BuildTime=${AK_BUILD_TIME}" \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="$BUILD_LDFLAGS" -o /out/ak-api ./cmd/ak-api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="$BUILD_LDFLAGS" -o /out/ak-cli ./cmd/ak-cli \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="$BUILD_LDFLAGS" -o /out/ak-worker ./cmd/ak-worker

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
