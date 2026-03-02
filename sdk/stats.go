package sdk

import "sort"

// AuthorRank holds contributor statistics.
type AuthorRank struct {
	Name    string
	Modules int
	CVEs    int
}

// Rankings returns a sorted leaderboard of exploit authors.
func Rankings() []AuthorRank {
	mu.RLock()
	defer mu.RUnlock()

	stats := make(map[string]*AuthorRank)
	for _, e := range entries {
		info := e.mod.Info()
		for _, author := range info.Authors {
			rank, ok := stats[author]
			if !ok {
				rank = &AuthorRank{Name: author}
				stats[author] = rank
			}
			rank.Modules++
			rank.CVEs += len(info.CVEs())
		}
	}

	result := make([]AuthorRank, 0, len(stats))
	for _, rank := range stats {
		result = append(result, *rank)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Modules != result[j].Modules {
			return result[i].Modules > result[j].Modules
		}
		return result[i].CVEs > result[j].CVEs
	})
	return result
}
