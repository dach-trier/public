FROM --platform=$BUILDPLATFORM golang:1.26.2-alpine3.23 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app

COPY go.mod go.sum ./
RUN  go mod download

COPY cmd/server/main.go ./cmd/server/

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o ./out/server ./cmd/server

FROM alpine:3.23
WORKDIR /app
COPY --from=build /app/out/server /app/server
ENTRYPOINT ["/app/server"]
