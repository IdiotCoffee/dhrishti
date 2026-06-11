.PHONY: ebpf dhrishti stop-dhrishti run-simulation help

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

# Simulation tunables (override on CLI):
#   make run-simulation DURATION=10m VIRTUAL_USERS=1000 CONNECTING_IPS=15
DURATION        ?= 4m
VIRTUAL_USERS   ?= 500
CONNECTING_IPS  ?=

help:
	@echo "Dhrishti Makefile"
	@echo ""
	@echo "  make ebpf              Build eBPF probes"
	@echo "  make dhrishti          Start Go engine + History API + frontend"
	@echo "  make stop-dhrishti     Stop background Dhrishti processes"
	@echo "  make run-simulation    Run load test against mock stack"
	@echo ""
	@echo "Simulation options:"
	@echo "  DURATION=10m           Test length (5s, 4m, 1h, ...)"
	@echo "  VIRTUAL_USERS=500      k6 virtual users"
	@echo "  CONNECTING_IPS=10      Distinct client IPs (optional)"

ebpf:
	$(MAKE) -C $(ROOT)/ebpf

dhrishti: ebpf
	@chmod +x $(ROOT)/scripts/start-dhrishti.sh
	@$(ROOT)/scripts/start-dhrishti.sh

stop-dhrishti:
	@-pkill -f "go run main.go" 2>/dev/null || true
	@-pkill -f "uvicorn main:app" 2>/dev/null || true
	@-pkill -f "vite" 2>/dev/null || true
	@echo "Stopped Dhrishti processes (if any were running)."

run-simulation:
	@chmod +x $(ROOT)/scripts/simulation.sh
	@DURATION=$(DURATION) VIRTUAL_USERS=$(VIRTUAL_USERS) CONNECTING_IPS=$(CONNECTING_IPS) \
		$(ROOT)/scripts/simulation.sh
