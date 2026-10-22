BINARY := gost-panel
PKG    := ./cmd/gost-panel
LDFLAGS := -s -w

.PHONY: build linux windows clean test vet

# 默认：构建当前平台（纯 Go，无 CGO）
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

# Linux 服务器交叉编译
linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64 $(PKG)

# Windows 构建
windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY).exe $(PKG)

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY).exe
