BPF_CLANG=clang
BPFTOOL?=bpftool
VMLINUX_BTF?=/sys/kernel/btf/vmlinux
BPF_ARCH?=x86

BPF_CFLAGS=-O2 -g \
-target bpf \
-D__TARGET_ARCH_$(BPF_ARCH)

PROBES=\
tcp_connect \
tcp_close \
tcp_accept \
tcp_state

all: vmlinux.h $(PROBES)

# Generate the CO-RE type definitions from the running kernel. This avoids
# requiring users to download or maintain a separate vmlinux.h file.
vmlinux.h:
	@command -v $(BPFTOOL) >/dev/null 2>&1 || { echo "bpftool is required to generate ebpf/vmlinux.h"; exit 1; }
	@test -r "$(VMLINUX_BTF)" || { echo "Kernel BTF not found at $(VMLINUX_BTF). Enable CONFIG_DEBUG_INFO_BTF or install a kernel that provides /sys/kernel/btf/vmlinux."; exit 1; }
	$(BPFTOOL) btf dump file "$(VMLINUX_BTF)" format c > $@.tmp
	mv $@.tmp $@

$(PROBES): vmlinux.h
	$(BPF_CLANG) $(BPF_CFLAGS) \
	-c $@.bpf.c \
	-o $@.bpf.o

clean:
	rm -f *.bpf.o vmlinux.h vmlinux.h.tmp

# basically converts the C code into a bytecode - to be run with the command
# "make" - in this dir. very cool concept, basically a prebuilt script for compilation.
