FROM --platform=$BUILDPLATFORM golang:1.22 AS build
ARG TARGETARCH
WORKDIR /usr/src/app
COPY *.go go.mod go.sum ./
COPY internal internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -tags netgo -ldflags -w -o bin/frp-auth-plugin-$TARGETARCH frp-auth-plugin.go

FROM alpine:latest
ARG TARGETARCH
COPY --from=build /usr/src/app/bin/frp-auth-plugin-$TARGETARCH /usr/local/bin/frp-auth-plugin
CMD frp-auth-plugin
