package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const baseURL = "http://localhost:3000"

type LinksResponse struct {
	VoterLinks []struct {
		Name string `json:"Name"`
		URL  string `json:"URL"`
	} `json:"voter_links"`
	AdminLinks struct {
		Results string `json:"results"`
		Links   string `json:"links"`
	} `json:"admin_links"`
}

func TestIntegration_MultipleVoters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup: Create DynamoDB client for cleanup
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-2"),
	)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	// Configure for local DynamoDB
	endpoint := "http://localhost:8000"
	dbClient := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = &endpoint
	})

	// Cleanup function to remove test data
	cleanup := func() {
		t.Log("Cleaning up test data...")
		for _, voter := range Voters {
			_, _ = dbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String("placepoll-votes"),
				Key: map[string]types.AttributeValue{
					"voter": &types.AttributeValueMemberS{Value: voter},
				},
			})
		}
	}

	// Clean up before test (in case of previous failures)
	cleanup()

	// Defer cleanup after test
	defer cleanup()

	// Generate admin token
	adminToken, err := EncryptVoterToken("admin")
	if err != nil {
		t.Fatalf("Failed to generate admin token: %v", err)
	}

	// Get voter links
	resp, err := http.Get(fmt.Sprintf("%s/links?t=%s", baseURL, adminToken))
	if err != nil {
		t.Fatalf("Failed to get links: %v", err)
	}
	defer resp.Body.Close()

	var links LinksResponse
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
		t.Fatalf("Failed to decode links: %v", err)
	}

	if len(links.VoterLinks) == 0 {
		t.Fatal("No voter links returned")
	}

	// Fix URLs for local testing (handler always returns https://)
	for i := range links.VoterLinks {
		links.VoterLinks[i].URL = strings.Replace(links.VoterLinks[i].URL, "https://", "http://", 1)
	}
	links.AdminLinks.Results = strings.Replace(links.AdminLinks.Results, "https://", "http://", 1)

	// Launch browser
	path, _ := launcher.LookPath()
	u := launcher.New().Bin(path).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	// Simulate 4 voters with random selections
	numVoters := 4
	if numVoters > len(links.VoterLinks) {
		numVoters = len(links.VoterLinks)
	}

	rand.Seed(time.Now().UnixNano())

	t.Logf("Simulating %d voters...", numVoters)

	for i := 0; i < numVoters; i++ {
		voterLink := links.VoterLinks[i]
		t.Logf("Voting as %s...", voterLink.Name)

		page := browser.MustPage(voterLink.URL)
		defer page.MustClose()

		// Wait for page to load
		page.MustWaitLoad()

		// Verify voter name appears
		greeting := page.MustElement(".subtitle").MustText()
		if !contains(greeting, voterLink.Name) {
			t.Errorf("Expected greeting to contain %s, got: %s", voterLink.Name, greeting)
		}

		// Count destinations
		destinations := page.MustElements(".destination")
		t.Logf("Found %d destinations", len(destinations))

		if len(destinations) != len(Destinations) {
			t.Errorf("Expected %d destinations, found %d", len(Destinations), len(destinations))
		}

		// Collect all destination names first to avoid stale element references
		allDestNames := []string{}
		for _, destElem := range destinations {
			destName := destElem.MustElement(".dest-name").MustText()
			allDestNames = append(allDestNames, destName)
		}

		// Randomly set scores for each destination
		for _, destName := range allDestNames {
			// Random score 1-5
			score := rand.Intn(5) + 1

			// Set slider value and update display using JavaScript
			// This is more reliable than trying to manipulate range inputs directly
			page.MustEval(fmt.Sprintf(`() => {
				const slider = document.getElementById('score_%s');
				slider.value = %d;
				updateScore('%s', %d);
			}`, destName, score, destName, score))

			t.Logf("  %s: score %d", destName, score)
		}

		// Randomly select 0-2 dealbreakers
		numDealbreakers := rand.Intn(3) // 0, 1, or 2
		if numDealbreakers > 0 {
			// Shuffle and pick first N as dealbreakers
			shuffledNames := make([]string, len(allDestNames))
			copy(shuffledNames, allDestNames)
			rand.Shuffle(len(shuffledNames), func(i, j int) {
				shuffledNames[i], shuffledNames[j] = shuffledNames[j], shuffledNames[i]
			})

			for k := 0; k < numDealbreakers; k++ {
				destName := shuffledNames[k]
				// Use JavaScript to click the checkbox since destination names may contain
				// spaces and periods which are invalid in CSS selectors
				page.MustEval(fmt.Sprintf(`() => {
					const checkbox = document.getElementById('dealbreaker_%s');
					checkbox.click();
				}`, destName))
				t.Logf("  Dealbreaker: %s", destName)
			}
		}

		// Submit vote
		submitBtn := page.MustElement("button[type=submit]")
		submitBtn.MustClick()

		// Wait for redirect to results page
		time.Sleep(2 * time.Second)

		t.Logf("Vote submitted successfully for %s", voterLink.Name)
	}

	// Fetch and verify results
	t.Log("Fetching results...")
	resultsResp, err := http.Get(links.AdminLinks.Results)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}
	defer resultsResp.Body.Close()

	if resultsResp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK from results, got %d", resultsResp.StatusCode)
	}

	// Parse results page (basic check)
	page := browser.MustPage(links.AdminLinks.Results)
	defer page.MustClose()
	page.MustWaitLoad()

	// Check that winner box exists
	winnerBox := page.MustElement(".winner-box")
	winner := winnerBox.MustElement(".winner-name").MustText()

	t.Logf("Winner: %s", winner)

	// Verify winner is one of our destinations
	found := false
	for _, dest := range Destinations {
		if dest == winner {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Winner %s is not in Destinations list", winner)
	}

	// Check that results table exists
	resultsTable := page.MustElement(".results-table")
	rows := resultsTable.MustElements("tbody tr")

	t.Logf("Results table has %d rows", len(rows))

	if len(rows) == 0 {
		t.Error("Results table should have at least one row")
	}

	// Verify results are sorted by score (descending)
	var prevScore int = 999999
	for _, row := range rows {
		cells := row.MustElements("td")
		if len(cells) >= 3 {
			scoreText := cells[2].MustText()
			var currentScore int
			fmt.Sscanf(scoreText, "%d", &currentScore)

			if currentScore > prevScore {
				t.Errorf("Results not sorted correctly: %d > %d", currentScore, prevScore)
			}
			prevScore = currentScore
		}
	}

	t.Log("Integration test completed successfully!")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
