build:
	go build -ldflags="-s -w -X main.version=develop"
pack:
	upx --best --lzma gai

