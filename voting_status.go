package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const votingStatusKey = "_voting_status"

type VotingStatus struct {
	IsOpen          bool
	ScheduledCutoff *time.Time // nil means no scheduled cutoff
	ClosedAt        *time.Time // nil means not closed yet
}

// GetVotingStatus retrieves the current voting status
func GetVotingStatus(ctx context.Context, client *dynamodb.Client) (*VotingStatus, error) {
	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("placepoll-votes"),
		Key: map[string]types.AttributeValue{
			"voter": &types.AttributeValueMemberS{Value: votingStatusKey},
		},
	})

	if err != nil {
		return nil, err
	}

	// If no status record exists, voting is open by default
	if result.Item == nil {
		return &VotingStatus{
			IsOpen:          true,
			ScheduledCutoff: nil,
			ClosedAt:        nil,
		}, nil
	}

	status := &VotingStatus{
		IsOpen: true,
	}

	// Parse IsOpen
	if v, ok := result.Item["is_open"]; ok {
		if boolVal, ok := v.(*types.AttributeValueMemberBOOL); ok {
			status.IsOpen = boolVal.Value
		}
	}

	// Parse ScheduledCutoff
	if v, ok := result.Item["scheduled_cutoff"]; ok {
		if strVal, ok := v.(*types.AttributeValueMemberS); ok && strVal.Value != "" {
			if t, err := time.Parse(time.RFC3339, strVal.Value); err == nil {
				status.ScheduledCutoff = &t
			}
		}
	}

	// Parse ClosedAt
	if v, ok := result.Item["closed_at"]; ok {
		if strVal, ok := v.(*types.AttributeValueMemberS); ok && strVal.Value != "" {
			if t, err := time.Parse(time.RFC3339, strVal.Value); err == nil {
				status.ClosedAt = &t
			}
		}
	}

	// Check if scheduled cutoff has passed
	if status.ScheduledCutoff != nil && time.Now().After(*status.ScheduledCutoff) {
		status.IsOpen = false
		// Update the record to mark as closed
		if status.ClosedAt == nil {
			now := time.Now()
			status.ClosedAt = &now
			_ = SaveVotingStatus(ctx, client, status)
		}
	}

	return status, nil
}

// SaveVotingStatus saves the voting status
func SaveVotingStatus(ctx context.Context, client *dynamodb.Client, status *VotingStatus) error {
	item := map[string]types.AttributeValue{
		"voter":   &types.AttributeValueMemberS{Value: votingStatusKey},
		"is_open": &types.AttributeValueMemberBOOL{Value: status.IsOpen},
	}

	if status.ScheduledCutoff != nil {
		item["scheduled_cutoff"] = &types.AttributeValueMemberS{
			Value: status.ScheduledCutoff.Format(time.RFC3339),
		}
	} else {
		item["scheduled_cutoff"] = &types.AttributeValueMemberS{Value: ""}
	}

	if status.ClosedAt != nil {
		item["closed_at"] = &types.AttributeValueMemberS{
			Value: status.ClosedAt.Format(time.RFC3339),
		}
	} else {
		item["closed_at"] = &types.AttributeValueMemberS{Value: ""}
	}

	_, err := client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("placepoll-votes"),
		Item:      item,
	})

	return err
}

// CloseVotingNow immediately closes voting
func CloseVotingNow(ctx context.Context, client *dynamodb.Client) error {
	now := time.Now()
	status := &VotingStatus{
		IsOpen:          false,
		ScheduledCutoff: nil,
		ClosedAt:        &now,
	}
	return SaveVotingStatus(ctx, client, status)
}

// SetScheduledCutoff sets a future cutoff time
func SetScheduledCutoff(ctx context.Context, client *dynamodb.Client, cutoffTime time.Time) error {
	status := &VotingStatus{
		IsOpen:          true,
		ScheduledCutoff: &cutoffTime,
		ClosedAt:        nil,
	}
	return SaveVotingStatus(ctx, client, status)
}

// ReopenVoting reopens voting
func ReopenVoting(ctx context.Context, client *dynamodb.Client) error {
	status := &VotingStatus{
		IsOpen:          true,
		ScheduledCutoff: nil,
		ClosedAt:        nil,
	}
	return SaveVotingStatus(ctx, client, status)
}
