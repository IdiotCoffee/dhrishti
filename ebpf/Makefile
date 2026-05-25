BPF_CLANG=clang

BPF_CFLAGS=-O2 -g \
-target bpf \
-D__TARGET_ARCH_x86

PROBES=\
tcp_connect \
tcp_close \
tcp_accept \
tcp_state

all: $(PROBES)

$(PROBES):
	$(BPF_CLANG) $(BPF_CFLAGS) \
	-c $@.bpf.c \
	-o $@.bpf.o

clean:
	rm -f *.bpf.o

# basically converts the C code into a bytecode - to be run with the command
# "make" - in this dir. very cool concept, basically a prebuilt script for compilation.
