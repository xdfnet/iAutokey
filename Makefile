VERSION := $(shell grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' package.json | head -1 | sed 's/.*: *"\(.*\)"/\1/')
SIGN_ID := 4A287668E97BC130AA6D19F4D64799394CAACBAD

BIN   := build/iautokey
DST   := $$HOME/.local/bin/iautokey
PLIST := $$HOME/Library/LaunchAgents/com.user.iautokey.plist

.PHONY: build test release clean help install sign deploy plist restart

help:
	@echo "iautokey $(VERSION)"
	@echo ""
	@echo "  make build      # 编译"
	@echo "  make install    # 安装到 ~/.local/bin"
	@echo "  make sign       # Developer ID 签名"
	@echo "  make deploy     # build + install + sign + restart"
	@echo "  make plist      # 配置开机自启"
	@echo "  make test       # 测试"
	@echo "  make release    # 发布到 npm + git tag"
	@echo "  make clean      # 清理"

build:
	@mkdir -p build
	@go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BIN) .
	@echo "编译完成: $(BIN)"

install: build
	@cp $(BIN) $(DST)
	@chmod +x $(DST)
	@echo "已安装: $(DST)"

sign:
	@codesign --remove-signature $(DST) 2>/dev/null; true
	@codesign -s $(SIGN_ID) -f --identifier com.user.iautokey $(DST)
	@echo "已签名"

deploy: install sign restart

restart:
	@launchctl unload $(PLIST) 2>/dev/null; true
	@sleep 0.3
	@launchctl load -w $(PLIST)
	@echo "已重启"
	@sleep 0.5
	@$(BIN) status

plist:
	@mkdir -p $$HOME/.config/iautokey
	@cp configs/com.user.iautokey.plist $(PLIST)
	@launchctl unload $(PLIST) 2>/dev/null; true
	@launchctl load -w $(PLIST)
	@echo "已配置开机自启"

test:
	@go test ./...
	@echo "ok"

release: test
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "工作区不干净，请先提交"; \
		git status --short; \
		exit 1; \
	fi
	@if npm view @xdfnet/iautokey@$(VERSION) version >/dev/null 2>&1; then \
		echo "npm 版本已存在: @xdfnet/iautokey@$(VERSION)"; \
		exit 1; \
	fi
	git tag v$(VERSION)
	git push origin HEAD
	git push origin v$(VERSION)
	npm publish --access public

clean:
	@rm -rf build
	@echo "清理完成"
