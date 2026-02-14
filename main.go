package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var dbClient *dynamodb.Client

func init() {
	// Initialize AWS SDK and DynamoDB client
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(fmt.Sprintf("failed to load AWS config: %v", err))
	}

	// Check if we should use a local DynamoDB endpoint
	if endpoint := os.Getenv("DYNAMODB_ENDPOINT"); endpoint != "" {
		dbClient = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = &endpoint
		})
	} else {
		dbClient = dynamodb.NewFromConfig(cfg)
	}
}

// Handler is the Lambda entry point
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Route based on method and path
	method := request.HTTPMethod
	path := request.Path

	switch {
	case method == "GET" && path == "/vote":
		return HandleVoteGet(ctx, dbClient, request)
	case method == "POST" && path == "/vote":
		return HandleVotePost(ctx, dbClient, request)
	case method == "GET" && path == "/results":
		return HandleResults(ctx, dbClient, request)
	case method == "GET" && path == "/links":
		return HandleLinks(ctx, dbClient, request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Not Found",
		}, nil
	}
}

func main() {
	lambda.Start(Handler)
}
