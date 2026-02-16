package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

//go:embed templates/*.html
var templatesFS embed.FS

// HandleVoteGet displays the voting form
func HandleVoteGet(ctx context.Context, dbClient *dynamodb.Client, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract token from query string
	token, ok := request.QueryStringParameters["t"]
	if !ok || token == "" {
		return errorResponse(http.StatusBadRequest, "Missing token parameter"), nil
	}

	// Decrypt token to get voter name
	voterName, err := DecryptVoterToken(token)
	if err != nil {
		return errorResponse(http.StatusUnauthorized, "Invalid token"), nil
	}

	// Verify voter is in authorized list
	if !isValidVoter(voterName) {
		return errorResponse(http.StatusUnauthorized, "Unauthorized voter"), nil
	}

	// Check if voter has already submitted
	existingVote, err := GetVote(ctx, dbClient, voterName)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Database error"), nil
	}

	// Render voting form template
	tmpl, err := template.ParseFS(templatesFS, "templates/vote.html")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Sprintf("Template error: %v", err)), nil
	}

	// Sort destinations alphabetically
	sortedDests := make([]string, len(Destinations))
	copy(sortedDests, Destinations)
	sort.Strings(sortedDests)

	data := map[string]interface{}{
		"Voter":        voterName,
		"Destinations": sortedDests,
		"ExistingVote": existingVote,
		"Token":        token,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return errorResponse(http.StatusInternalServerError, "Template execution error"), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: buf.String(),
	}, nil
}

// HandleVotePost processes vote submission
func HandleVotePost(ctx context.Context, dbClient *dynamodb.Client, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Check if voting is open
	votingStatus, err := GetVotingStatus(ctx, dbClient)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to check voting status"), nil
	}

	if !votingStatus.IsOpen {
		return errorResponse(http.StatusForbidden, "Voting has been closed"), nil
	}

	// Parse form data
	formData, err := url.ParseQuery(request.Body)
	if err != nil {
		return errorResponse(http.StatusBadRequest, "Invalid form data"), nil
	}

	// Extract and verify token
	token := formData.Get("token")
	if token == "" {
		return errorResponse(http.StatusBadRequest, "Missing token"), nil
	}

	voterName, err := DecryptVoterToken(token)
	if err != nil {
		return errorResponse(http.StatusUnauthorized, "Invalid token"), nil
	}

	if !isValidVoter(voterName) {
		return errorResponse(http.StatusUnauthorized, "Unauthorized voter"), nil
	}

	// Parse scores
	scores := make(map[string]int)
	for _, dest := range Destinations {
		scoreStr := formData.Get("score_" + dest)
		if scoreStr == "" {
			continue
		}
		score, err := strconv.Atoi(scoreStr)
		if err != nil || score < 1 || score > 5 {
			return errorResponse(http.StatusBadRequest, fmt.Sprintf("Invalid score for %s", dest)), nil
		}
		scores[dest] = score
	}

	// Parse dealbreakers
	var dealbreakers []string
	for _, dest := range Destinations {
		if formData.Get("dealbreaker_"+dest) == "on" {
			dealbreakers = append(dealbreakers, dest)
		}
	}

	// Validate dealbreaker limit
	if len(dealbreakers) > 2 {
		return errorResponse(http.StatusBadRequest, "You may only select up to 2 dealbreakers"), nil
	}

	// Save vote
	vote := &Vote{
		Voter:        voterName,
		Scores:       scores,
		Dealbreakers: dealbreakers,
	}

	if err := SaveVote(ctx, dbClient, vote); err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to save vote"), nil
	}

	// Show confirmation page
	tmpl, err := template.ParseFS(templatesFS, "templates/confirmation.html")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Template error"), nil
	}

	data := map[string]interface{}{
		"Voter": voterName,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return errorResponse(http.StatusInternalServerError, "Template execution error"), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: buf.String(),
	}, nil
}

// HandleResults displays the voting results
func HandleResults(ctx context.Context, dbClient *dynamodb.Client, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Require admin token
	token, ok := request.QueryStringParameters["t"]
	if !ok || token == "" {
		return errorResponse(http.StatusUnauthorized, "Admin token required"), nil
	}

	// Decrypt and verify admin token
	userName, err := DecryptVoterToken(token)
	if err != nil {
		return errorResponse(http.StatusUnauthorized, "Invalid token"), nil
	}

	if !isAdminUser(userName) {
		return errorResponse(http.StatusForbidden, "Admin access required"), nil
	}

	// Fetch all votes
	votes, err := ListAllVotes(ctx, dbClient)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to fetch votes"), nil
	}

	// Calculate results
	results := TallyResults(votes)
	winner := ""
	if len(results) > 0 {
		winner = results[0].Name
	}

	// Collect all dealbreakers
	eliminatedDestinations := make(map[string]bool)
	for _, vote := range votes {
		for _, db := range vote.Dealbreakers {
			eliminatedDestinations[db] = true
		}
	}

	// Build complete results including eliminated destinations
	type VoterScore struct {
		Voter        string
		Score        int
		IsDealbreaker bool
	}
	type RankedResult struct {
		Rank        int
		Name        string
		Score       int
		VoterScores []VoterScore
		Eliminated  bool
	}

	// Add valid results first
	rankedResults := make([]RankedResult, 0)
	for i, r := range results {
		// Collect individual voter scores for this destination
		voterScores := make([]VoterScore, 0)
		for _, vote := range votes {
			if score, ok := vote.Scores[r.Name]; ok {
				// Check if this destination is a dealbreaker for this voter
				isDealbreaker := false
				for _, db := range vote.Dealbreakers {
					if db == r.Name {
						isDealbreaker = true
						break
					}
				}

				voterScores = append(voterScores, VoterScore{
					Voter:        vote.Voter,
					Score:        score,
					IsDealbreaker: isDealbreaker,
				})
			}
		}

		rankedResults = append(rankedResults, RankedResult{
			Rank:        i + 1,
			Name:        r.Name,
			Score:       r.Score,
			VoterScores: voterScores,
			Eliminated:  false,
		})
	}

	// Add eliminated destinations at the end
	for dest := range eliminatedDestinations {
		// Skip if already in results (shouldn't happen, but safety check)
		found := false
		for _, r := range rankedResults {
			if r.Name == dest {
				found = true
				break
			}
		}
		if found {
			continue
		}

		// Calculate score and collect voter details for eliminated destination
		totalScore := 0
		voterScores := make([]VoterScore, 0)
		for _, vote := range votes {
			if score, ok := vote.Scores[dest]; ok {
				totalScore += score
				isDealbreaker := false
				for _, db := range vote.Dealbreakers {
					if db == dest {
						isDealbreaker = true
						break
					}
				}
				voterScores = append(voterScores, VoterScore{
					Voter:        vote.Voter,
					Score:        score,
					IsDealbreaker: isDealbreaker,
				})
			}
		}

		rankedResults = append(rankedResults, RankedResult{
			Rank:        0, // No rank for eliminated
			Name:        dest,
			Score:       totalScore,
			VoterScores: voterScores,
			Eliminated:  true,
		})
	}

	// Sort eliminated destinations by score (descending) for consistent ordering
	sort.SliceStable(rankedResults, func(i, j int) bool {
		// Keep non-eliminated at top, sort eliminated by score
		if rankedResults[i].Eliminated == rankedResults[j].Eliminated {
			if rankedResults[i].Eliminated {
				return rankedResults[i].Score > rankedResults[j].Score
			}
			return rankedResults[i].Rank < rankedResults[j].Rank
		}
		return !rankedResults[i].Eliminated
	})

	// Render results template
	tmpl, err := template.ParseFS(templatesFS, "templates/results.html")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Template error"), nil
	}

	data := map[string]interface{}{
		"Results":    rankedResults,
		"Winner":     winner,
		"TotalVotes": len(votes),
		"Voters":     Voters,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return errorResponse(http.StatusInternalServerError, "Template execution error"), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: buf.String(),
	}, nil
}

// HandleLinks generates encrypted voter links
func HandleLinks(ctx context.Context, dbClient *dynamodb.Client, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Require admin token
	token, ok := request.QueryStringParameters["t"]
	if !ok || token == "" {
		return errorResponse(http.StatusUnauthorized, "Admin token required"), nil
	}

	// Decrypt and verify admin token
	userName, err := DecryptVoterToken(token)
	if err != nil {
		return errorResponse(http.StatusUnauthorized, "Invalid token"), nil
	}

	if !isAdminUser(userName) {
		return errorResponse(http.StatusForbidden, "Admin access required"), nil
	}

	// Get base URL from request
	host := request.Headers["Host"]
	if host == "" {
		host = request.Headers["host"] // try lowercase
	}

	// Custom domain (placepoll.cyou) doesn't need stage in path
	// API Gateway direct access includes /Prod/ stage
	baseURL := "https://" + host
	if strings.Contains(host, "execute-api") {
		// Direct API Gateway access - include stage
		baseURL = fmt.Sprintf("%s/%s", baseURL, request.RequestContext.Stage)
	}

	// Generate links for all voters
	type VoterLink struct {
		Name string
		URL  string
	}

	type LinksResponse struct {
		VoterLinks  []VoterLink `json:"voter_links"`
		AdminLinks  map[string]string `json:"admin_links"`
	}

	voterLinks := make([]VoterLink, 0, len(Voters))
	for _, voter := range Voters {
		token, err := EncryptVoterToken(voter)
		if err != nil {
			return errorResponse(http.StatusInternalServerError, "Failed to generate token"), nil
		}

		link := VoterLink{
			Name: voter,
			URL:  fmt.Sprintf("%s/vote?t=%s", baseURL, url.QueryEscape(token)),
		}
		voterLinks = append(voterLinks, link)
	}

	// Generate admin token for results/links access
	adminToken, err := EncryptVoterToken("admin")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to generate admin token"), nil
	}

	response := LinksResponse{
		VoterLinks: voterLinks,
		AdminLinks: map[string]string{
			"results": fmt.Sprintf("%s/results?t=%s", baseURL, url.QueryEscape(adminToken)),
			"links":   fmt.Sprintf("%s/links?t=%s", baseURL, url.QueryEscape(adminToken)),
		},
	}

	// Return as JSON
	body, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to generate response"), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

// HandleAdmin displays the admin dashboard
func HandleAdmin(ctx context.Context, dbClient *dynamodb.Client, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract and verify admin token
	token, ok := request.QueryStringParameters["t"]
	if !ok || token == "" {
		return errorResponse(http.StatusBadRequest, "Missing token parameter"), nil
	}

	userName, err := DecryptVoterToken(token)
	if err != nil {
		return errorResponse(http.StatusUnauthorized, "Invalid token"), nil
	}

	if !isAdminUser(userName) {
		return errorResponse(http.StatusForbidden, "Admin access required"), nil
	}

	// Get voting status
	votingStatus, err := GetVotingStatus(ctx, dbClient)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to get voting status"), nil
	}

	// Get base URL
	host := request.Headers["Host"]
	if host == "" {
		host = request.Headers["host"]
	}
	baseURL := "https://" + host
	if strings.Contains(host, "execute-api") {
		baseURL = fmt.Sprintf("%s/%s", baseURL, request.RequestContext.Stage)
	}

	// Generate admin token
	adminToken, err := EncryptVoterToken("admin")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Failed to generate admin token"), nil
	}

	// Generate voter links
	type VoterLink struct {
		Name string
		URL  string
	}
	voterLinks := make([]VoterLink, 0, len(Voters))
	for _, voter := range Voters {
		voterToken, err := EncryptVoterToken(voter)
		if err != nil {
			continue
		}
		voterLinks = append(voterLinks, VoterLink{
			Name: voter,
			URL:  fmt.Sprintf("%s/vote?t=%s", baseURL, url.QueryEscape(voterToken)),
		})
	}

	// Prepare template data
	data := map[string]interface{}{
		"VotingStatus": votingStatus,
		"Token":        token,
		"ResultsURL":   fmt.Sprintf("%s/results?t=%s", baseURL, url.QueryEscape(adminToken)),
		"LinksURL":     fmt.Sprintf("%s/links?t=%s", baseURL, url.QueryEscape(adminToken)),
		"AdminURL":     fmt.Sprintf("%s/admin?t=%s", baseURL, url.QueryEscape(token)),
		"VoterLinks":   voterLinks,
	}

	// Render template
	tmpl, err := template.ParseFS(templatesFS, "templates/admin.html")
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "Template error"), nil
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return errorResponse(http.StatusInternalServerError, "Template execution error"), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: buf.String(),
	}, nil
}

// HandleAdminAction processes admin actions (close voting, schedule cutoff, reopen)
func HandleAdminAction(ctx context.Context, dbClient *dynamodb.Client, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse form data
	formData, err := url.ParseQuery(request.Body)
	if err != nil {
		return errorResponse(http.StatusBadRequest, "Invalid form data"), nil
	}

	// Verify admin token
	token := formData.Get("token")
	if token == "" {
		return errorResponse(http.StatusBadRequest, "Missing token"), nil
	}

	userName, err := DecryptVoterToken(token)
	if err != nil {
		return errorResponse(http.StatusUnauthorized, "Invalid token"), nil
	}

	if !isAdminUser(userName) {
		return errorResponse(http.StatusForbidden, "Admin access required"), nil
	}

	// Get action
	action := formData.Get("action")
	switch action {
	case "close_now":
		if err := CloseVotingNow(ctx, dbClient); err != nil {
			return errorResponse(http.StatusInternalServerError, "Failed to close voting"), nil
		}
	case "schedule_cutoff":
		cutoffTimeStr := formData.Get("cutoff_time")
		if cutoffTimeStr == "" {
			return errorResponse(http.StatusBadRequest, "Missing cutoff_time"), nil
		}
		cutoffTime, err := time.Parse("2006-01-02T15:04", cutoffTimeStr)
		if err != nil {
			return errorResponse(http.StatusBadRequest, "Invalid cutoff_time format"), nil
		}
		if err := SetScheduledCutoff(ctx, dbClient, cutoffTime); err != nil {
			return errorResponse(http.StatusInternalServerError, "Failed to schedule cutoff"), nil
		}
	case "reopen":
		if err := ReopenVoting(ctx, dbClient); err != nil {
			return errorResponse(http.StatusInternalServerError, "Failed to reopen voting"), nil
		}
	default:
		return errorResponse(http.StatusBadRequest, "Invalid action"), nil
	}

	// Redirect back to admin page
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusSeeOther,
		Headers: map[string]string{
			"Location": fmt.Sprintf("/admin?t=%s", url.QueryEscape(token)),
		},
	}, nil
}

// Helper functions

func isValidVoter(name string) bool {
	for _, voter := range Voters {
		if voter == name {
			return true
		}
	}
	return false
}

func isAdminUser(name string) bool {
	for _, admin := range AdminUsers {
		if admin == name {
			return true
		}
	}
	return false
}

func errorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: message,
	}
}
