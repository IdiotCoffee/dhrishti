package history

import (
	"fmt"
	"strings"
)

const (
	servicesKnownKey  = "dhrishti:services:known"
	edgesKnownKey     = "dhrishti:edges:known"
	snapshotsIndexKey = "dhrishti:snapshots:index"
)

func sanitizeLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(":", "_", "|", "_", " ", "_")
	return replacer.Replace(value)
}

func edgeSeriesKey(source, dest, metric string) string {
	return fmt.Sprintf(
		"dhrishti:ts:edge:%s:%s:%s",
		sanitizeLabel(source),
		sanitizeLabel(dest),
		metric,
	)
}

func serviceSeriesKey(service, metric string) string {
	return fmt.Sprintf(
		"dhrishti:ts:svc:%s:%s",
		sanitizeLabel(service),
		metric,
	)
}

func snapshotKey(tsMs int64) string {
	return fmt.Sprintf("dhrishti:snapshot:%d", tsMs)
}

func edgeKnownMember(source, dest string) string {
	return sanitizeLabel(source) + "|" + sanitizeLabel(dest)
}
