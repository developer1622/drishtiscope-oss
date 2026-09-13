.PHONY: all ui mock generate run build clean demo help test test-race vulncheck

BINARY   := drishtiscope
BACKEND  := ./backend
FRONTEND := ./frontend
DIST     := ./frontend/dist
PORT     ?= 8080
COMM     ?= agy
DB       ?= drishtiscope.db

all: help

help:
	@echo ""
	@echo "  DrishtiScope (दृष्टिScope) — Real-Time Linux & AI Agent Kernel Observability"
	@echo ""
	@echo "  make demo          Mock backend + React UI (no root needed)"
	@echo "  make mock          Go backend in MOCK mode only"
	@echo "  make ui            Vite dev server (frontend only)"
	@echo "  make run           AUTO mode (tries eBPF, falls back to mock)"
	@echo "  make build         Production binary + Vite bundle"
	@echo "  make generate      Compile eBPF C → embedded object"
	@echo "  make test          Go unit tests + frontend typecheck"
	@echo "  make clean         Remove build artifacts"
	@echo ""
	@echo "  Target process comm prefix: COMM=$(COMM)  (override: make run COMM=agy)"
	@echo "  SQLite TSDB path:           DB=$(DB)"
	@echo ""

demo:
	@echo "→ Starting DrishtiScope in DEMO mode..."
	@echo "  Backend: http://localhost:$(PORT)"
	@echo "  UI:      http://localhost:5173"
	@$(MAKE) -j2 _demo-backend _demo-ui

_demo-backend:
	cd $(BACKEND) && MODE=mock HTTP_ADDR=:$(PORT) TARGET_COMM=$(COMM) DB_PATH=../$(DB) go run ./cmd/agentscope

_demo-ui:
	cd $(FRONTEND) && npm run dev

mock:
	cd $(BACKEND) && MODE=mock HTTP_ADDR=:$(PORT) TARGET_COMM=$(COMM) DB_PATH=../$(DB) go run ./cmd/agentscope

ui:
	cd $(FRONTEND) && npm run dev

run:
	@echo "→ AUTO mode (tries eBPF, falls back to mock)"
	@echo "   Real eBPF needs: root or CAP_BPF+CAP_PERFMON+CAP_SYS_ADMIN"
	cd $(BACKEND) && MODE=auto HTTP_ADDR=:$(PORT) TARGET_COMM=$(COMM) DB_PATH=../$(DB) go run ./cmd/agentscope

generate:
	@echo "→ Compiling eBPF with clang..."
	clang -g -O2 -target bpf -D__TARGET_ARCH_x86 \
		-I/usr/include/$$(uname -m)-linux-gnu -I/usr/include \
		-c $(BACKEND)/bpf/agent.bpf.c \
		-o $(BACKEND)/bpf/agent.bpf.o
	cp $(BACKEND)/bpf/agent.bpf.o $(BACKEND)/internal/ebpfagent/agent.bpf.o
	@echo "→ Embedded object updated: $(BACKEND)/internal/ebpfagent/agent.bpf.o"

test:
	cd $(BACKEND) && go test ./...
	cd $(FRONTEND) && npx tsc --noEmit

test-race:
	cd $(BACKEND) && go test -race ./...

vulncheck:
	cd $(BACKEND) && go run golang.org/x/vuln/cmd/govulncheck@latest ./...

build: build-frontend build-backend

build-frontend:
	@echo "→ Building React frontend..."
	cd $(FRONTEND) && npm run build

build-backend:
	@echo "→ Building Go binary..."
	cd $(BACKEND) && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ../$(BINARY) ./cmd/agentscope
	ln -sf $(BINARY) agentscope
	@echo "→ Binary: ./$(BINARY) (symlinked to ./agentscope)"
	@echo "   sudo ./$(BINARY) --mode=ebpf --comm=$(COMM) --db=$(DB)"

clean:
	rm -f $(BINARY) agentscope
	rm -rf $(DIST)
	@echo "→ Clean done"
