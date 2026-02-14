# PlacePoll

A serverless ranked voting application for choosing travel destinations among friends.

## Overview

PlacePoll allows a group of friends to vote on travel destinations using a ranked scoring system (1-5) with optional dealbreakers. Each voter receives a secure, encrypted link to cast their vote privately. An admin can view aggregated results to determine the winning destination.

**Live Demo:** https://placepoll.cyou

## Features

- **Ranked Voting:** Score each destination from 1-5 based on preference
- **Dealbreakers:** Mark up to 2 destinations as absolute no-gos (eliminates them from results)
- **Secure Authentication:** Token-based authentication using AES-256-GCM encryption
- **Tie-Breaking:** Automatic alphabetical ordering for destinations with equal scores
- **Real-time Results:** Admin dashboard showing ranked results and winner
- **Serverless Architecture:** Fully serverless with automatic scaling

## Tech Stack

- **Backend:** Go 1.25.7
- **Runtime:** AWS Lambda (provided.al2023)
- **API:** AWS API Gateway with custom domain
- **Database:** DynamoDB (serverless NoSQL)
- **Infrastructure:** AWS SAM (Serverless Application Model)
- **DNS:** Route53 with ACM certificate
- **Testing:** Go standard testing + Rod browser automation

## Architecture

```
┌─────────────┐
│   Route53   │ placepoll.cyou
└──────┬──────┘
       │
┌──────▼──────────────┐
│  API Gateway        │
│  Custom Domain      │
└──────┬──────────────┘
       │
┌──────▼──────────────┐
│  Lambda Function    │
│  (Go Handler)       │
└──────┬──────────────┘
       │
┌──────▼──────────────┐
│  DynamoDB Table     │
│  placepoll-votes    │
└─────────────────────┘
```

## Project Structure

```
placepoll/
├── main.go              # Lambda entry point and router
├── config.go            # Destinations, voters, and encryption key
├── crypto.go            # AES-256-GCM token encryption/decryption
├── db.go                # DynamoDB operations
├── tally.go             # Vote tallying and winner calculation
├── handlers.go          # HTTP request handlers
├── templates/
│   ├── vote.html        # Voting form interface
│   └── results.html     # Results dashboard
├── template.yaml        # AWS SAM infrastructure definition
├── Taskfile.yml         # Task runner commands
├── docker-compose.yml   # Local DynamoDB for testing
└── cmd/
    └── gentoken/        # Admin token generator utility
```

## Local Development

### Prerequisites

- Go 1.25.7+
- Docker and Docker Compose
- AWS SAM CLI
- [Task](https://taskfile.dev/) runner

### Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd placepoll
```

2. Copy environment template:
```bash
cp env.example.json env.json
```

3. Start local development environment:
```bash
task local:start
```

This starts DynamoDB Local and the Lambda function at http://localhost:3000

### Generate Admin Token

To access the admin endpoints (/results, /links):

```bash
task generate-admin-token
```

Copy the generated token and use it in URLs:
- Links: http://localhost:3000/links?t=YOUR_TOKEN
- Results: http://localhost:3000/results?t=YOUR_TOKEN

### Stop Local Environment

```bash
task local:stop
```

## Deployment

### Prerequisites

- AWS CLI configured with appropriate credentials
- AWS SAM CLI installed
- Domain registered (optional, for custom domain)

### First Deployment

1. Build the application:
```bash
task build
```

2. Deploy with guided prompts:
```bash
task deploy:guided
```

Follow the prompts to configure:
- Stack name: placepoll
- AWS Region: us-east-2 (or your preferred region)
- Confirm changes before deploy: Y
- Allow SAM CLI IAM role creation: Y
- Disable rollback: N
- Save arguments to configuration file: Y

### Subsequent Deployments

```bash
task deploy
```

### Custom Domain Setup

The template includes Route53, ACM certificate, and API Gateway custom domain configuration. After deployment:

1. Note the nameservers from the CloudFormation outputs
2. Update your domain registrar's DNS settings to use the AWS nameservers
3. Wait for DNS propagation (can take up to 48 hours)
4. Access your app at https://your-domain.com

## Usage

### For Admins

1. Generate an admin token locally:
```bash
task generate-admin-token
```

2. Access the links endpoint:
```
https://placepoll.cyou/links?t=YOUR_ADMIN_TOKEN
```

3. Distribute voter links to participants

4. View results:
```
https://placepoll.cyou/results?t=YOUR_ADMIN_TOKEN
```

### For Voters

1. Receive your unique voting link from the admin
2. Open the link in your browser
3. Score each destination (1-5) using the sliders
4. Optionally mark up to 2 dealbreakers
5. Submit your vote
6. View the results (redirected automatically)

## Testing

### Unit Tests

Run the comprehensive tally logic tests:

```bash
go test -v -run TestTally
```

Test coverage includes:
- No votes scenario
- Single voter
- Multiple voters
- Dealbreaker elimination
- Tie-breaking logic
- Edge cases (all eliminated)

### Integration Tests

Run end-to-end browser automation tests:

```bash
go test -v -run TestIntegration
```

This simulates 4 voters making random selections and validates:
- Vote submission workflow
- JavaScript slider interactions
- Dealbreaker selection
- Results page rendering
- Winner calculation

Skip integration tests:
```bash
go test -short
```

## Configuration

### Voters and Destinations

Edit `config.go` to customize:

```go
var Voters = []string{
    "Alice",
    "Bob",
    "Carol",
    // Add more voters...
}

var Destinations = []string{
    "Chicago",
    "Denver",
    // Add more destinations...
}
```

### Encryption Key

**IMPORTANT:** Change the AES encryption key in `config.go` before deployment:

```go
var AESKey = []byte("your-32-byte-key-here-change-this")
```

The key must be exactly 32 bytes for AES-256.

## Voting Logic

1. **Scoring:** Each voter assigns scores (1-5) to destinations
2. **Dealbreakers:** Voters can veto up to 2 destinations
3. **Tallying:**
   - Sum all scores for each destination
   - Eliminate destinations marked as dealbreakers by any voter
   - Sort by total score (descending)
   - Tie-breaker: Alphabetical order
4. **Winner:** Highest-scoring non-eliminated destination

## Security

- **Token-based Auth:** All voter and admin links use AES-256-GCM encrypted tokens
- **HTTPS Only:** All traffic encrypted via ACM certificate
- **Stateless:** No sessions, all authentication via cryptographic tokens
- **Vote Privacy:** Individual votes stored but only admin can view aggregated results

## Available Tasks

View all available tasks:
```bash
task --list
```

Common tasks:
- `task build` - Build the Lambda function
- `task deploy` - Deploy to AWS
- `task local:start` - Start local development environment
- `task local:stop` - Stop local services
- `task generate-admin-token` - Generate admin authentication token

## Troubleshooting

### DynamoDB Connection Issues

If local testing fails with DynamoDB errors:
```bash
task local:stop
task local:start
```

### Template Validation Errors

Validate the SAM template:
```bash
sam validate --lint
```

### Lambda Build Failures

Ensure Go modules are up to date:
```bash
go mod tidy
go mod download
```

## License

MIT License - Feel free to use this for your friend group's travel planning!

## Contributing

This is a personal project for coordinating travel among friends. Feel free to fork and adapt for your own use cases.
