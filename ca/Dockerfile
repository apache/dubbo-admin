FROM golang:latest as builder

RUN apk --update add gcc libc-dev upx ca-certificates && update-ca-certificates

ADD . /workspace

WORKDIR /workspace

RUN --mount=type=cache,target=/go \
  go build -buildmode=pie -ldflags "-linkmode external -extldflags -static -w" \
  -o /workspace/ca

FROM alpine:3.17

EXPOSE 30060
EXPOSE 30062

COPY --from=builder /workspace/ca /ca

CMD ["/ca"]
