SIGN_ID := 4A287668E97BC130AA6D19F4D64799394CAACBAD
DST     := $$HOME/.local/bin/iAutokey
PLIST   := $$HOME/Library/LaunchAgents/com.xdfnet.iAutokey.plist

VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
BIN     := build/iAutokey

.PHONY: build install sign deploy plist help

help:
	@echo "iAutokey $(VERSION)"
	@echo ""
	@echo "  make build       # 编译"
	@echo "  make install     # 安装到 ~/.local/bin"
	@echo "  make sign        # 签名（首次必须，之后仅重建时需重签）"
	@echo "  make plist       # 安装 LaunchAgent（开机自启）"
	@echo "  make deploy      # 编译+签名+安装+重启"
	@echo "  make clean       # 清理"
	@echo ""
	@echo "首次使用:"
	@echo "  1. make deploy"
	@echo "  2. 去 系统设置→隐私与安全性→辅助功能 添加 iAutokey"
	@echo "  3. make plist    # 配置开机自启"

build:
	@mkdir -p build
	@go build -ldflags="-s -w" -o $(BIN) .
	@echo "编译完成: $(BIN)"

install: build
	@cp $(BIN) $(DST)
	@chmod +x $(DST)
	@echo "已安装: $(DST)"

sign:
	@codesign --remove-signature $(DST) 2>/dev/null; true
	@codesign -s $(SIGN_ID) -f --identifier com.xdfnet.iAutokey $(DST)
	@echo "已签名"

deploy: install sign restart

plist:
	@cp configs/com.xdfnet.iAutokey.plist $(PLIST) 2>/dev/null || cp /dev/null $(PLIST)
	@launchctl unload $(PLIST) 2>/dev/null; true
	@launchctl load $(PLIST)
	@echo "已配置开机自启"

restart:
	@launchctl unload $(PLIST) 2>/dev/null; true
	@sleep 0.3
	@launchctl load $(PLIST)
	@echo "已重启"

clean:
	@rm -rf build
	@echo "清理完成"
