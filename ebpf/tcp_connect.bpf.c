// a minimal EBPF probe that will:
// 1. hook to tcp_connect()
// 2. extract the metadata
// 3. emit an event...

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "GPL";

// create a struct to store an event:
// it has the process ID, the communicator, the destination address and the destination port.
struct event {
    __u32 pid;
    char comm[16];
    __u32 daddr;
    __u16 dport;
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF); // creates a high performance event channel.
    __uint(max_entries, 1 << 24);
} events SEC(".maps");

// below code basically says: run this program whenever tcp_connect() executes.
SEC("kprobe/tcp_connect")
int trace_tcp_connect(struct pt_regs *ctx)
{
    struct sock *sk;
    struct event *e;

    sk = (struct sock *)PT_REGS_PARM1(ctx);

    e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);

    if (!e) {
        return 0;
    }

    // gets the PID of the host - kernel ALWAYS sees real host PIDs.
    e->pid = bpf_get_current_pid_tgid() >> 32;

    // gets the process name - python, curl, nginx,...
    bpf_get_current_comm(&e->comm, sizeof(e->comm));

    bpf_probe_read_kernel(
        &e->daddr,
        sizeof(e->daddr),
        &sk->__sk_common.skc_daddr
    );

    __u16 dport;

    bpf_probe_read_kernel(
        &dport,
        sizeof(dport),
        &sk->__sk_common.skc_dport
    );

    e->dport = __builtin_bswap16(dport);

    bpf_ringbuf_submit(e, 0);

    return 0;
}
