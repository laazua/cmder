# 项目打包

agent := cmd-agent
proxy := cmd-proxy
agent_bin := bin/agent/$(agent)
proxy_bin := bin/proxy/$(proxy)

.PHONY: clean build agent proxy

# 设置静态编译
CLIB_ENABLE := $(shell go env CGO_ENABLED)
ifeq ($(CLIB_ENABLE),1)
	go env -w CGO_ENABLED=0
endif

build: $(agent_bin) $(proxy_bin)

$(agent_bin): cmd/agent/*.go
	go build -C cmd/agent -o $(agent)
	@if [ ! -d bin/agent ]; then \
		mkdir -p bin/agent; \
	fi
	cp cmd/agent/$(agent) $(agent_bin)
	@if [ -f docs/agent.yaml ]; then \
		cp docs/agent.yaml bin/agent/config.yaml; \
	fi

$(proxy_bin): cmd/proxy/*.go
	go build -C cmd/proxy -o $(proxy)
	@if [ ! -d bin/proxy ]; then \
		mkdir -p bin/proxy; \
	fi
	cp cmd/proxy/$(proxy) $(proxy_bin)
	@if [ -f docs/proxy.yaml ]; then \
		cp docs/proxy.yaml bin/proxy/config.yaml; \
	fi

	@if [ ! -d bin/proxy/web ]; then \
	    mkdir -p bin/proxy/web; \
		cp web/index.html bin/proxy/web; \
	fi
	
agent: $(agent_bin)
	@echo "Starting agent... (Press Ctrl+C to stop)"
	@trap '' INT; cd bin/agent && ./$(agent) || true

proxy: $(proxy_bin)
	@echo "Starting proxy... (Press Ctrl+C to stop)"
	@trap '' INT; cd bin/proxy && ./$(proxy) || true

clean:
	rm -f cmd/agent/$(agent)
	rm -f cmd/proxy/$(proxy)
	rm -rf bin