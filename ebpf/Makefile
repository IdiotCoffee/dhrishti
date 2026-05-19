BPF_CLANG=clang
BPF_CFLAGS=-O2 -g -target bpf -D__TARGET_ARCH_x86

all:
	$(BPF_CLANG) $(BPF_CFLAGS) \
	-c tcp_connect.bpf.c \
	-o tcp_connect.bpf.o

# basically converts the C code into a bytecode - to be run with the command
# "make" - in this dir. very cool concept, basically a prebuilt script for compilation.
