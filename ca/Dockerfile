FROM golang:1.20.1-alpine3.17 as builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk --update add gcc libc-dev upx ca-certificates && update-ca-certificates

ADD . /workspace

WORKDIR /workspace

RUN --mount=type=cache,target=/go \
  env GOPROXY=https://goproxy.cn,direct \
  go build -buildmode=pie -ldflags "-linkmode external -extldflags -static -w" \
  -o /workspace/ca

FROM alpine:3.17

EXPOSE 30060
EXPOSE 30062

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

COPY --from=builder /workspace/ca /ca

CMD ["/ca"]
