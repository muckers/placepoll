package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const TableName = "placepoll-votes"

// Vote represents a voter's submission
type Vote struct {
	Voter        string         `dynamodbav:"voter"`
	Scores       map[string]int `dynamodbav:"scores"`        // destination -> score (1-5)
	Dealbreakers []string       `dynamodbav:"dealbreakers"`  // list of vetoed destinations
	SubmittedAt  string         `dynamodbav:"submitted_at"`  // ISO 8601 timestamp
}

// SaveVote stores a vote in DynamoDB
func SaveVote(ctx context.Context, client *dynamodb.Client, vote *Vote) error {
	// Set timestamp if not already set
	if vote.SubmittedAt == "" {
		vote.SubmittedAt = time.Now().UTC().Format(time.RFC3339)
	}

	item, err := attributevalue.MarshalMap(vote)
	if err != nil {
		return fmt.Errorf("failed to marshal vote: %w", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save vote: %w", err)
	}

	return nil
}

// GetVote retrieves a vote by voter name
func GetVote(ctx context.Context, client *dynamodb.Client, voter string) (*Vote, error) {
	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"voter": &types.AttributeValueMemberS{Value: voter},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get vote: %w", err)
	}

	if result.Item == nil {
		return nil, nil // Vote not found
	}

	var vote Vote
	err = attributevalue.UnmarshalMap(result.Item, &vote)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal vote: %w", err)
	}

	return &vote, nil
}

// ListAllVotes retrieves all votes from DynamoDB
func ListAllVotes(ctx context.Context, client *dynamodb.Client) ([]*Vote, error) {
	result, err := client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(TableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan votes: %w", err)
	}

	votes := make([]*Vote, 0, len(result.Items))
	for _, item := range result.Items {
		var vote Vote
		err = attributevalue.UnmarshalMap(item, &vote)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal vote: %w", err)
		}
		votes = append(votes, &vote)
	}

	return votes, nil
}
