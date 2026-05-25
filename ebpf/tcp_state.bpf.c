#include "vmlinux.h"

#define AF_INET 2

#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_tracing.h>

#include "common.h"

char LICENSE[] SEC("license") = "GPL";

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} events SEC(".maps");

SEC("kprobe/tcp_set_state")
int BPF_KPROBE(trace_tcp_set_state,
               struct sock *sk,
               int state)
{
    struct tcp_state_event *event;

    __u16 family;

    int oldstate;

    if (!sk)
        return 0;

    family = BPF_CORE_READ(sk, __sk_common.skc_family);

    if (family != AF_INET)
        return 0;

    oldstate = BPF_CORE_READ(sk, __sk_common.skc_state);

    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);

    if (!event)
        return 0;

    event->pid = bpf_get_current_pid_tgid() >> 32;

    bpf_get_current_comm(&event->comm, sizeof(event->comm));

    event->saddr = BPF_CORE_READ(sk, __sk_common.skc_rcv_saddr);

    event->daddr = BPF_CORE_READ(sk, __sk_common.skc_daddr);

    event->sport = BPF_CORE_READ(sk, __sk_common.skc_num);

    event->dport = BPF_CORE_READ(sk, __sk_common.skc_dport);

    event->dport = __builtin_bswap16(event->dport);

    event->old_state = oldstate;

    event->new_state = state;

    event->timestamp_ns = bpf_ktime_get_ns();

    bpf_ringbuf_submit(event, 0);

    return 0;
}
