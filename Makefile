# 项目打包

agent := cmd-agent
proxy := cmd-proxy
agent_bin := bin/agent/$(agent)
proxy_bin := bin/proxy/$(proxy)
build_args := -ldflags="-w -s" -trimpath

.PHONY: clean build agent proxy vendor

# 设置静态编译
CLIB_ENABLE := $(shell go env CGO_ENABLED)
ifeq ($(CLIB_ENABLE),1)
	go env -w CGO_ENABLED=0
endif

# 设置平台
GO_PLATFORM := $(shell go env GOOS)
ifneq ($(GO_PLATFORM), linux)
    go env -w GOOS=linux
endif
# 设置架构
GO_ARCH := $(shell go env GOARCH)
ifneq ($(GO_ARCH), amd64)
    go env -w GOARCH=amd64
endif

# 在 build 目标前添加 vendor 依赖
build: $(agent_bin) $(proxy_bin)

# vendor 目标：创建 vendor 目录
vendor:
	go mod vendor
	@echo "Vendor directory created/updated"

$(agent_bin): cmd/agent/*.go
	go build -C cmd/agent -o $(agent) $(build_args)
	@if [ ! -d bin/agent ]; then \
		mkdir -p bin/agent; \
	fi
	cp cmd/agent/$(agent) $(agent_bin)
	@if [ -f docs/agent.yaml ]; then \
		cp docs/agent.yaml bin/agent/config.yaml; \
	fi

$(proxy_bin): cmd/proxy/*.go
	go build -C cmd/proxy -o $(proxy) $(build_args)
	@if [ ! -d bin/proxy ]; then \
		mkdir -p bin/proxy; \
	fi
	cp cmd/proxy/$(proxy) $(proxy_bin)
	@if [ -f docs/proxy.yaml ]; then \
		cp docs/proxy.yaml bin/proxy/config.yaml; \
	fi

	@if [ ! -d bin/proxy/web ]; then \
	    mkdir -p bin/proxy/web; \
		cp cmd/proxy/web/index.html bin/proxy/web; \
	fi
	
agent: $(agent_bin)
	@echo "Starting agent... (Press Ctrl+C to stop)"
	@trap '' INT; cd bin/agent && ./$(agent) || true

proxy: $(proxy_bin)
	@echo "Starting proxy... (Press Ctrl+C to stop)"
	@trap '' INT; cd bin/proxy && ./$(proxy) || true

clean:
	rm -rf bin vendor
	rm -f cmd/agent/$(agent)
	rm -f cmd/proxy/$(proxy)
