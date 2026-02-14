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

### Built-in Prerequisite Checking

The Taskfile automatically verifies prerequisites before running tasks. If a requirement is not met, you'll receive a clear error message with instructions on how to fix it.

**What gets checked automatically:**

| Task | Prerequisite Checked | Error Message Guidance |
|------|---------------------|------------------------|
| `admin:init` | `jq` command installed | Install command provided |
| `generate-admin-token` | `AES_KEY` env var set | Suggests running `admin:init` |
| `deploy` | `samconfig.toml` exists | Directs to `deploy:guided` |
| `deploy` | Runs `build` first | Automatic dependency |
| `deploy:guided` | Runs `build` first | Automatic dependency |
| `local:start` | `env.json` exists | Suggests running `admin:init` |
| `local:start` | Runs `local:db` first | Automatic dependency |
| `local:db` | Docker running | Clear Docker startup message |
| `local:create-table` | DynamoDB on port 8000 | Directs to `local:db` |

**Benefits:**
- No silent failures - you'll know immediately what's missing
- Helpful error messages guide you to the solution
- Automatic task dependencies ensure correct execution order
- Reduces setup friction for new developers

### Administration & Setup

#### `task admin:init`
**Initialize admin setup (first-time setup)**

Complete initialization for a new installation:
1. Generates a new AES-256 encryption key
2. Creates/updates `env.json` with the new key
3. Generates an admin token for accessing protected endpoints

Use this when setting up the project for the first time or when you need to rotate the encryption key.

```bash
task admin:init
```

**Prerequisites:**
- `jq` command-line JSON processor (checked automatically)
  - Install: `sudo apt install jq` (Linux) or `brew install jq` (macOS)

**Note:** This is typically the first command you run after cloning the repo.

#### `task generate-key`
**Generate a new AES-256 encryption key**

Creates a cryptographically secure 32-byte key for AES-256-GCM encryption. Outputs the key in base64 format for use in environment variables.

```bash
task generate-key
```

**Output:** Displays the generated key in multiple formats (raw base64, export statement, JSON format).

**Use case:** When you need to manually generate a new key without updating env.json, or when rotating keys for production deployment.

#### `task generate-admin-token`
**Generate an admin authentication token**

Creates an encrypted token that grants access to admin-only endpoints (`/results` and `/links`). Requires `AES_KEY` environment variable to be set.

```bash
export AES_KEY="your-base64-key"
task generate-admin-token
```

**Prerequisites:**
- `AES_KEY` environment variable must be set (checked automatically)
- The key must match what's deployed (for production) or in env.json (for local)

**Output:** Admin token and example URLs for accessing protected endpoints.

**Note:** If `AES_KEY` is not set, the task will fail with a helpful error message. Run `task admin:init` to set up everything automatically.

### Build & Deployment

#### `task build`
**Build the Lambda function**

Compiles the Go application and prepares it for deployment using AWS SAM. Creates the `.aws-sam` directory with build artifacts.

```bash
task build
```

**Prerequisites:**
- Go 1.25+ installed
- AWS SAM CLI installed

**What it does:**
- Compiles Go code to a `bootstrap` binary (required for custom Lambda runtime)
- Packages the binary with AWS Lambda Go runtime
- Embeds HTML templates into the binary

**When to use:** Before deploying, or when testing the build locally. SAM automatically builds before deployment, so this is optional.

#### `task deploy:guided`
**Deploy with guided prompts (first-time deployment)**

Interactive deployment wizard that walks through all configuration options. Use this for the first deployment to AWS.

```bash
task deploy:guided
```

**Prerequisites:**
- AWS CLI configured with credentials
- AWS SAM CLI installed
- Generated AES key (run `task generate-key` first)

**Automatically runs:** `task build` (enforced via task dependency)

**What it prompts for:**
- Stack name (default: `placepoll`)
- AWS Region (e.g., `us-east-2`)
- **AES_KEY parameter** (your base64-encoded encryption key)
- Confirmation before creating/updating resources
- IAM role creation permissions
- Save configuration to samconfig.toml

**Duration:** 5-10 minutes for initial deployment (includes Route53, ACM certificate, custom domain setup).

#### `task deploy`
**Deploy to AWS (subsequent deployments)**

Fast deployment using saved configuration from previous `deploy:guided` run. Uses settings stored in `samconfig.toml`.

```bash
task deploy
```

**Prerequisites:**
- `samconfig.toml` must exist (checked automatically)
- Must have run `deploy:guided` at least once

**Automatically runs:** `task build` (enforced via task dependency)

**Duration:** 2-3 minutes for updates (infrastructure already exists).

**What it deploys:**
- Lambda function with latest code changes
- Updates to infrastructure (if template.yaml changed)
- Environment variables (including AES_KEY if changed)

**Note:** If `samconfig.toml` is missing, the task will fail with a message to run `task deploy:guided` first.

#### `task validate`
**Validate the SAM template**

Checks `template.yaml` for syntax errors and CloudFormation compatibility issues.

```bash
task validate
```

**Use case:** Run before deployment to catch template errors early, or during development when modifying infrastructure.

### Local Development

#### `task local:start`
**Start local development environment**

Launches both DynamoDB Local and the Lambda function API on your machine. This is the main command for local development.

```bash
task local:start
```

**Prerequisites:**
- `env.json` must exist with valid `AES_KEY` (checked automatically)
  - Run `task admin:init` if this file doesn't exist
- Docker must be running (checked automatically via `local:db` dependency)

**Automatically runs:** `task local:db` (enforced via task dependency)

**What it does:**
1. Calls `local:db` to start DynamoDB Local in Docker
2. Creates the `placepoll-votes` table in local DynamoDB
3. Starts SAM local API server on http://localhost:3000

**Endpoints available:**
- `http://localhost:3000/vote?t=TOKEN` - Voting form
- `http://localhost:3000/results?t=ADMIN_TOKEN` - Results page
- `http://localhost:3000/links?t=ADMIN_TOKEN` - Generate voter links

**Note:** This command blocks the terminal. Use Ctrl+C to stop, then run `task local:stop` to clean up.

#### `task local:db`
**Start DynamoDB Local (without Lambda)**

Starts only the DynamoDB Local container and creates the votes table. Useful for database-only testing.

```bash
task local:db
```

**Prerequisites:**
- Docker must be running (checked automatically)
- Docker Compose installed

**What it does:**
1. Creates `env.json` from `env.example.json` if it doesn't exist
2. Starts DynamoDB Local container on port 8000
3. Creates network `placepoll-net` for inter-container communication
4. Creates `placepoll-votes` table

**Use case:** When you want to test database operations independently, or prepare the database before starting the Lambda function.

**Note:** If Docker is not running, the task will fail with a helpful error message.

#### `task local:create-table`
**Create the DynamoDB table in local database**

Standalone command to create the votes table in an already-running DynamoDB Local instance.

```bash
task local:create-table
```

**Prerequisites:**
- DynamoDB Local must be running on port 8000 (checked automatically)
- Run `task local:db` first if not already running

**Use case:**
- When the table was deleted and needs recreation
- For manual database setup in testing scenarios

**Note:** If DynamoDB Local is not accessible on port 8000, the task will fail with a message to run `task local:db` first.

#### `task local:stop`
**Stop local development environment**

Stops and removes all local Docker containers (DynamoDB Local) and networks.

```bash
task local:stop
```

**What it does:**
- Stops `placepoll-dynamodb` container
- Removes the container
- Removes `placepoll-net` network

**Note:** This does NOT stop the SAM local API (if running in another terminal). Use Ctrl+C in that terminal to stop SAM.

### Database Operations

#### Viewing Production Database State

To inspect votes in the production DynamoDB table:

**View all votes:**
```bash
export AWS_PROFILE=your-profile-name
aws dynamodb scan --table-name placepoll-votes --region us-east-2
```

**Get a specific voter's vote:**
```bash
export AWS_PROFILE=your-profile-name
aws dynamodb get-item \
  --table-name placepoll-votes \
  --key '{"voter":{"S":"Lesley"}}' \
  --region us-east-2
```

**Count total votes:**
```bash
export AWS_PROFILE=your-profile-name
aws dynamodb scan \
  --table-name placepoll-votes \
  --select COUNT \
  --region us-east-2
```

**Delete a specific vote (cleanup/testing):**
```bash
export AWS_PROFILE=your-profile-name
aws dynamodb delete-item \
  --table-name placepoll-votes \
  --key '{"voter":{"S":"Lesley"}}' \
  --region us-east-2
```

**Delete all votes (complete reset):**
```bash
export AWS_PROFILE=your-profile-name
for voter in Lesley Casey James Rebecca Kate Monica Aaron; do
  aws dynamodb delete-item \
    --table-name placepoll-votes \
    --key '{"voter":{"S":"'$voter'"}}' \
    --region us-east-2
done
```

**Prerequisites:**
- AWS CLI configured with appropriate profile
- Profile must have DynamoDB read/write permissions
- Correct region specified (check your deployment region)

**Note:** Replace `your-profile-name` with your actual AWS CLI profile name and adjust the region if you deployed to a different region than `us-east-2`.

#### Viewing Local Database State

To inspect votes in local DynamoDB during development:

**View all votes:**
```bash
aws dynamodb scan \
  --table-name placepoll-votes \
  --endpoint-url http://localhost:8000 \
  --region us-east-1
```

**Get a specific voter's vote:**
```bash
aws dynamodb get-item \
  --table-name placepoll-votes \
  --key '{"voter":{"S":"Lesley"}}' \
  --endpoint-url http://localhost:8000 \
  --region us-east-1
```

**Delete all votes (local cleanup):**
```bash
for voter in Lesley Casey James Rebecca Kate Monica Aaron; do
  aws dynamodb delete-item \
    --table-name placepoll-votes \
    --key '{"voter":{"S":"'$voter'"}}' \
    --endpoint-url http://localhost:8000 \
    --region us-east-1
done
```

**Prerequisites:**
- DynamoDB Local must be running (`task local:db` or `task local:start`)

### Utility & Maintenance

#### `task clean`
**Clean build artifacts**

Removes all build artifacts created by SAM and Go compilation.

```bash
task clean
```

**What it removes:**
- `.aws-sam/` directory (SAM build cache)
- `bootstrap` binary (compiled Lambda function)

**Use case:**
- When switching branches or resolving build issues
- To force a complete rebuild from scratch
- Before committing (though these are already gitignored)

#### `task logs`
**Tail CloudWatch logs for the Lambda function**

Streams real-time logs from your deployed Lambda function in AWS.

```bash
task logs
```

**Prerequisites:**
- Application must be deployed to AWS
- AWS CLI configured with appropriate permissions

**Use case:**
- Debugging production issues
- Monitoring live traffic
- Viewing error traces from failed requests

**Note:** Use Ctrl+C to stop tailing logs.

#### `task delete`
**Delete the CloudFormation stack**

Completely removes all AWS resources created by this application (Lambda, DynamoDB, API Gateway, Route53, ACM certificate, custom domain).

```bash
task delete
```

**Warning:** This is destructive and irreversible. All votes and configuration will be deleted.

**What it removes:**
- Lambda function
- DynamoDB table (all votes lost)
- API Gateway
- Custom domain configuration
- Route53 hosted zone
- ACM certificate
- CloudWatch logs (optional, will be prompted)

**Use case:**
- Cleaning up after testing
- Decommissioning the application
- Before redeploying with a different stack name

#### `task links`
**Information about generating voter links**

Displays instructions for generating voter links after deployment.

```bash
task links
```

**Output:** Reminder that voter links are generated by calling the `/links` endpoint with an admin token.

**Actual usage:**
```bash
# After deployment, use the admin token to get links
curl "https://placepoll.cyou/links?t=YOUR_ADMIN_TOKEN"
```

#### `task help`
**Show all available tasks**

Displays a list of all tasks with their descriptions. Equivalent to `task --list`.

```bash
task help
```

## Task Workflow Examples

### First-Time Setup (Local)
```bash
# 1. Initialize admin (generates key, updates env.json, creates admin token)
#    Automatically checks for jq installation
task admin:init

# 2. Start local environment
#    Automatically checks Docker is running and env.json exists
#    Automatically runs local:db first
task local:start

# 3. In another terminal, generate voter links
#    The admin token was already generated in step 1
# Use the token to access http://localhost:3000/links?t=TOKEN
```

**Prerequisite checks performed automatically:**
- Step 1: Verifies `jq` is installed
- Step 2: Verifies Docker is running, `env.json` exists with AES_KEY

### First-Time Deployment (AWS)
```bash
# 1. Generate encryption key
task generate-key
# Save the output - you'll need it for deployment

# 2. Build and deploy with guided prompts
#    Automatically builds the application first
task deploy:guided
# Enter the key when prompted for AES_KEY parameter

# 3. Update nameservers at your domain registrar
# (use NS records from CloudFormation outputs)

# 4. Generate admin token for production
export AES_KEY="your-production-key"
#    Automatically checks AES_KEY is set before running
task generate-admin-token

# 5. Get voter links
curl "https://placepoll.cyou/links?t=YOUR_ADMIN_TOKEN"
```

**Prerequisite checks performed automatically:**
- Step 2: Automatically runs `task build` before deploying
- Step 4: Verifies `AES_KEY` environment variable is set

### Making Code Changes
```bash
# 1. Make your changes to *.go files

# 2. Test locally
#    Automatically checks Docker and env.json
task local:start
# Test your changes at http://localhost:3000

# 3. Deploy to AWS
#    Automatically builds and checks samconfig.toml exists
task deploy

# 4. Monitor logs if needed
task logs
```

**Prerequisite checks performed automatically:**
- Step 2: Verifies Docker is running and env.json exists
- Step 3: Runs `task build` and verifies samconfig.toml exists

### Rotating Encryption Keys
```bash
# 1. Generate new key
task generate-key

# 2. Update local env.json manually with new key

# 3. Redeploy with new key
task deploy:guided
# Enter the new key when prompted

# 4. Generate new admin token
export AES_KEY="new-key"
task generate-admin-token

# Note: Old voter links will be invalid after key rotation
```

### Cleaning Up
```bash
# Stop local services
task local:stop

# Clean build artifacts
task clean

# Delete from AWS (if no longer needed)
task delete
```

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
