package config

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

type Config struct {
	EntryServices []string `json:"entry_services"`
}

func Load() Config {
	cfg := Config{
		EntryServices: []string{"gateway"},
	}

	path := os.Getenv("DHRISHTI_CONFIG")
	if path == "" {
		path = "dhrishti.json"
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Printf("[config] parsing %s: %v", path, err)
		}
	}

	if env := os.Getenv("DHRISHTI_ENTRY_SERVICES"); env != "" {
		parts := strings.Split(env, ",")
		cfg.EntryServices = cfg.EntryServices[:0]
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				cfg.EntryServices = append(cfg.EntryServices, s)
			}
		}
	}

	log.Printf("[config] entry services: %v", cfg.EntryServices)
	return cfg
}

func (c Config) IsEntryService(name string) bool {
	for _, entry := range c.EntryServices {
		if entry == name {
			return true
		}
	}
	return false
}
