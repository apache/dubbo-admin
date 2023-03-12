# Build the image binary
FROM golang:1.20.1-alpine3.17 as builder


# Build argments
ARG LDFLAGS
ARG PKGNAME
ARG BUILD

WORKDIR /go/src/github.com/apache/dubbo-admin

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

#RUN if [[ "${PKGNAME}" == "authority" ]]; then apk --update add gcc libc-dev upx ca-certificates && update-ca-certificates; fi

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN if [[ "${BUILD}" != "CI" ]]; then go env -w GOPROXY=https://goproxy.cn,direct; fi
RUN go env
RUN go mod download

# Copy the go source
COPY pkg pkg/
COPY cmd cmd/

# Build
RUN env
RUN go build -ldflags="${LDFLAGS}" -a -o ${PKGNAME} /go/src/github.com/apache/dubbo-admin/cmd/${PKGNAME}/main.go


FROM alpine:3.17
WORKDIR /
ARG PKGNAME
COPY --from=builder /go/src/github.com/apache/dubbo-admin/${PKGNAME} .

