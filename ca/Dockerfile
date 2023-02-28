FROM golang:1.20.1 as builder

RUN apt-get update && apt-get install -y gcc libc-dev upx ca-certificates && update-ca-certificates

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
