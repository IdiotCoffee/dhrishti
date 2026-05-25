#include "vmlinux.h"
#include "common.h"
#include <bpf/bpf_tracing.h>

#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
// #include <linux/in.h>

#define AF_INET 2
char LICENSE[] SEC("license") = "GPL";

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} events SEC(".maps");


// run when a connection closes
SEC("kprobe/tcp_close")
int trace_tcp_close(struct pt_regs *ctx)
{
    struct sock *sk;
    struct tcp_close_event *event;

    __u16 family;
// extract the socket for the closing connection
    sk = (struct sock *)PT_REGS_PARM1(ctx);

    if (!sk)
        return 0;
// filter IPv4
    family = BPF_CORE_READ(sk, __sk_common.skc_family);

    if (family != AF_INET)
        return 0;

    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);

    if (!event)
        return 0;

// I want to create some space inside the kernel ring buffer (basically a mailbox)
    event->pid = bpf_get_current_pid_tgid() >> 32;

    bpf_get_current_comm(&event->comm, sizeof(event->comm));

    event->saddr = BPF_CORE_READ(sk, __sk_common.skc_rcv_saddr);

    event->daddr = BPF_CORE_READ(sk, __sk_common.skc_daddr);

    event->sport = BPF_CORE_READ(sk, __sk_common.skc_num);

    event->dport = BPF_CORE_READ(sk, __sk_common.skc_dport);

    event->dport = __builtin_bswap16(event->dport);
    // get the time stamp
    event->timestamp_ns = bpf_ktime_get_ns();
    // emit the event
    bpf_ringbuf_submit(event, 0);

    return 0;
}
