package types

import (
	"strings"
	"sync"
	"time"
)

type UnknownIPEntry struct {
	IP                string
	ConnectionCount   int
	ActiveConnections int
	Destinations      map[string]int
	LastSeen          time.Time
}

type UnknownIPRegistry struct {
	Mu      sync.RWMutex
	Entries map[string]*UnknownIPEntry
}

func NewUnknownIPRegistry() *UnknownIPRegistry {
	return &UnknownIPRegistry{
		Entries: make(map[string]*UnknownIPEntry),
	}
}

func IsLoopbackIP(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "127.")
}

func (r *UnknownIPRegistry) RecordConnect(ip, destination string) {
	if ip == "" || IsLoopbackIP(ip) {
		return
	}

	now := time.Now()

	r.Mu.Lock()
	defer r.Mu.Unlock()

	entry, exists := r.Entries[ip]
	if !exists {
		entry = &UnknownIPEntry{
			IP:           ip,
			Destinations: make(map[string]int),
		}
		r.Entries[ip] = entry
	}

	entry.ConnectionCount++
	entry.ActiveConnections++
	entry.Destinations[destination]++
	entry.LastSeen = now
}

func (r *UnknownIPRegistry) RecordClose(ip, destination string) {
	if ip == "" || IsLoopbackIP(ip) {
		return
	}

	r.Mu.Lock()
	defer r.Mu.Unlock()

	entry, exists := r.Entries[ip]
	if !exists {
		return
	}

	if entry.ActiveConnections > 0 {
		entry.ActiveConnections--
	}

	entry.LastSeen = time.Now()
}
