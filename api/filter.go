package api

func isHiddenGraphNode(name string) bool {
	switch name {
	case "host", "unknown", "external":
		return true
	default:
		return false
	}
}

func shouldShowEdge(source, target string) bool {
	if isHiddenGraphNode(source) || isHiddenGraphNode(target) {
		return false
	}
	return true
}
