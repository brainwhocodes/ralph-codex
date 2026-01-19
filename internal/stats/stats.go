package stats

// StatsProvider is implemented by components that can provide statistics
type StatsProvider interface {
	GetStats() map[string]interface{}
}

// Aggregate collects stats from multiple providers into a single map
func Aggregate(providers ...StatsProvider) map[string]interface{} {
	result := make(map[string]interface{})
	for _, p := range providers {
		if p != nil {
			for k, v := range p.GetStats() {
				result[k] = v
			}
		}
	}
	return result
}
