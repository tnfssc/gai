FROM vborja/asdf-alpine AS builder

COPY ./.tool-versions ./
RUN asdf plugin add golang && asdf plugin add upx && asdf install

WORKDIR /build
COPY ./go.mod ./go.sum ./main.go ./
RUN go build -ldflags="-s -w -X main.version=unknown"
RUN upx --best --lzma --force-macos gai*


FROM alpine
ENV SHELL sh
COPY --from=builder /build/gai .
ENTRYPOINT ["./gai"]
