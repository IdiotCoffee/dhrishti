#include "vmlinux.h"

#ifndef __COMMON_H__
#define __COMMON_H__
// #include <linux/types.h>

#define TASK_COMM_LEN 16

struct tcp_close_event {
    __u32 pid;

    char comm[TASK_COMM_LEN];

    __u32 saddr;
    __u32 daddr;

    __u16 sport;
    __u16 dport;

    __u64 timestamp_ns;
};

struct tcp_accept_event {
    __u32 pid;

    char comm[TASK_COMM_LEN];

    __u32 saddr;
    __u32 daddr;

    __u16 sport;
    __u16 dport;

    __u64 timestamp_ns;
};

struct tcp_state_event {
    __u32 pid;

    char comm[TASK_COMM_LEN];

    __u32 saddr;
    __u32 daddr;

    __u16 sport;
    __u16 dport;

    __u32 old_state;
    __u32 new_state;

    __u64 timestamp_ns;
};

struct tcp_connect_event {
    __u32 pid;

    char comm[TASK_COMM_LEN];

    __u32 saddr;
    __u32 daddr;

    __u16 sport;
    __u16 dport;

    __u64 timestamp_ns;
};

#endif
