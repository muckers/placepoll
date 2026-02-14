package main

import (
	"sort"
)

// DestinationResult represents a destination with its total score
type DestinationResult struct {
	Name  string
	Score int
}

// TallyResults calculates final scores after eliminating dealbreakers
func TallyResults(votes []*Vote) []DestinationResult {
	// Step 1: Collect all dealbreakers
	dealbreakers := make(map[string]bool)
	for _, vote := range votes {
		for _, dest := range vote.Dealbreakers {
			dealbreakers[dest] = true
		}
	}

	// Step 2: Calculate scores for non-dealbreaker destinations
	scores := make(map[string]int)
	for _, dest := range Destinations {
		// Skip destinations that are dealbreakers
		if dealbreakers[dest] {
			continue
		}

		// Sum up scores from all votes
		totalScore := 0
		for _, vote := range votes {
			if score, ok := vote.Scores[dest]; ok {
				totalScore += score
			}
		}
		scores[dest] = totalScore
	}

	// Step 3: Convert to sorted slice
	results := make([]DestinationResult, 0, len(scores))
	for dest, score := range scores {
		results = append(results, DestinationResult{
			Name:  dest,
			Score: score,
		})
	}

	// Sort by score descending (highest first), then alphabetically for ties
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Name < results[j].Name
		}
		return results[i].Score > results[j].Score
	})

	return results
}

// GetWinner returns the winning destination (highest score after dealbreaker elimination)
// Returns empty string if no valid destinations remain
func GetWinner(votes []*Vote) string {
	results := TallyResults(votes)
	if len(results) == 0 {
		return ""
	}
	return results[0].Name
}
