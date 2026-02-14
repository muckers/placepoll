package main

import (
	"testing"
)

func TestTallyResults_NoVotes(t *testing.T) {
	votes := []*Vote{}
	results := TallyResults(votes)

	if len(results) != len(Destinations) {
		t.Errorf("Expected %d destinations, got %d", len(Destinations), len(results))
	}

	// All destinations should have 0 score
	for _, r := range results {
		if r.Score != 0 {
			t.Errorf("Expected 0 score for %s, got %d", r.Name, r.Score)
		}
	}
}

func TestTallyResults_SingleVote(t *testing.T) {
	votes := []*Vote{
		{
			Voter: "Alice",
			Scores: map[string]int{
				"Chicago":  5,
				"Denver":   4,
				"Austin":   3,
				"Memphis":  2,
				"Milwaukee": 1,
			},
			Dealbreakers: []string{},
		},
	}

	results := TallyResults(votes)

	// Check Chicago is first (highest score)
	if results[0].Name != "Chicago" || results[0].Score != 5 {
		t.Errorf("Expected Chicago with score 5, got %s with score %d", results[0].Name, results[0].Score)
	}

	// Check Denver is second
	if results[1].Name != "Denver" || results[1].Score != 4 {
		t.Errorf("Expected Denver with score 4, got %s with score %d", results[1].Name, results[1].Score)
	}
}

func TestTallyResults_DealbreakersEliminate(t *testing.T) {
	votes := []*Vote{
		{
			Voter: "Alice",
			Scores: map[string]int{
				"Chicago": 5,
				"Denver":  4,
				"Austin":  3,
			},
			Dealbreakers: []string{"Chicago"}, // Eliminate highest scorer
		},
		{
			Voter: "Bob",
			Scores: map[string]int{
				"Chicago": 5,
				"Denver":  3,
				"Austin":  2,
			},
			Dealbreakers: []string{},
		},
	}

	results := TallyResults(votes)

	// Chicago should be eliminated despite having highest total score (10)
	for _, r := range results {
		if r.Name == "Chicago" {
			t.Errorf("Chicago should be eliminated by dealbreaker, but appears in results")
		}
	}

	// Denver should win with score 7
	if results[0].Name != "Denver" || results[0].Score != 7 {
		t.Errorf("Expected Denver to win with score 7, got %s with score %d", results[0].Name, results[0].Score)
	}
}

func TestTallyResults_TieBreaker(t *testing.T) {
	votes := []*Vote{
		{
			Voter: "Alice",
			Scores: map[string]int{
				"Milwaukee": 5,
				"Austin":    5,
				"Chicago":   5,
			},
			Dealbreakers: []string{},
		},
	}

	results := TallyResults(votes)

	// With equal scores (5), should be alphabetically ordered
	// Austin < Chicago < Milwaukee
	if results[0].Name != "Austin" {
		t.Errorf("Expected Austin to win tie (alphabetically), got %s", results[0].Name)
	}
	if results[1].Name != "Chicago" {
		t.Errorf("Expected Chicago in 2nd place, got %s", results[1].Name)
	}
	if results[2].Name != "Milwaukee" {
		t.Errorf("Expected Milwaukee in 3rd place, got %s", results[2].Name)
	}
}

func TestTallyResults_MultipleVoters(t *testing.T) {
	votes := []*Vote{
		{
			Voter: "Alice",
			Scores: map[string]int{
				"Chicago":  5,
				"Denver":   4,
				"Austin":   3,
				"Memphis":  2,
			},
			Dealbreakers: []string{},
		},
		{
			Voter: "Bob",
			Scores: map[string]int{
				"Chicago":  4,
				"Denver":   5,
				"Austin":   3,
				"Memphis":  1,
			},
			Dealbreakers: []string{},
		},
		{
			Voter: "Carol",
			Scores: map[string]int{
				"Chicago":  5,
				"Denver":   3,
				"Austin":   4,
				"Memphis":  2,
			},
			Dealbreakers: []string{},
		},
	}

	results := TallyResults(votes)

	// Chicago: 5+4+5 = 14
	// Denver: 4+5+3 = 12
	// Austin: 3+3+4 = 10
	// Memphis: 2+1+2 = 5

	if results[0].Name != "Chicago" || results[0].Score != 14 {
		t.Errorf("Expected Chicago with 14, got %s with %d", results[0].Name, results[0].Score)
	}
	if results[1].Name != "Denver" || results[1].Score != 12 {
		t.Errorf("Expected Denver with 12, got %s with %d", results[1].Name, results[1].Score)
	}
}

func TestTallyResults_MultipleDealbreakers(t *testing.T) {
	votes := []*Vote{
		{
			Voter: "Alice",
			Scores: map[string]int{
				"Chicago":  5,
				"Denver":   4,
				"Austin":   3,
			},
			Dealbreakers: []string{"Chicago"},
		},
		{
			Voter: "Bob",
			Scores: map[string]int{
				"Chicago":  5,
				"Denver":   4,
				"Austin":   2,
			},
			Dealbreakers: []string{"Denver"},
		},
	}

	results := TallyResults(votes)

	// Both Chicago and Denver eliminated
	for _, r := range results {
		if r.Name == "Chicago" || r.Name == "Denver" {
			t.Errorf("%s should be eliminated, but appears in results", r.Name)
		}
	}

	// Austin should win with score 5
	if results[0].Name != "Austin" || results[0].Score != 5 {
		t.Errorf("Expected Austin with 5, got %s with %d", results[0].Name, results[0].Score)
	}
}

func TestGetWinner(t *testing.T) {
	votes := []*Vote{
		{
			Voter: "Alice",
			Scores: map[string]int{
				"Chicago": 5,
				"Denver":  3,
			},
			Dealbreakers: []string{},
		},
	}

	winner := GetWinner(votes)
	if winner != "Chicago" {
		t.Errorf("Expected Chicago to win, got %s", winner)
	}
}

func TestGetWinner_AllEliminated(t *testing.T) {
	// Edge case: what if all destinations are dealbreakers?
	dealbreakers := make([]string, len(Destinations))
	copy(dealbreakers, Destinations)

	votes := []*Vote{
		{
			Voter:        "Alice",
			Scores:       map[string]int{"Chicago": 5},
			Dealbreakers: dealbreakers,
		},
	}

	winner := GetWinner(votes)
	if winner != "" {
		t.Errorf("Expected no winner when all eliminated, got %s", winner)
	}
}
