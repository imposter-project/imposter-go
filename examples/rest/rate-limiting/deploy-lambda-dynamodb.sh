#!/usr/bin/env bash
set -e

# Rate limiting example Lambda deployment with DynamoDB
# This script creates a DynamoDB table for rate limiting and deploys the application as a Lambda function

DYNAMODB_TABLE_NAME="imposter-rate-limiter"
LAMBDA_FUNCTION_NAME="imposter-rate-limiting"
LAMBDA_EXECUTION_ROLE="arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/ImposterLambdaExecutionRole"

# Check if AWS_REGION is set
if [ -z "${AWS_REGION}" ]; then
    echo "Warning: AWS_REGION environment variable is not set. AWS commands may fail or use default region."
fi

# Check if the required IAM role exists and update its policy
ROLE_NAME=$(echo "$LAMBDA_EXECUTION_ROLE" | sed 's/.*role\///')
echo "Checking if IAM role '$ROLE_NAME' exists"
if ! aws iam get-role --role-name "$ROLE_NAME" > /dev/null 2>&1; then
    echo "Error: IAM role '$ROLE_NAME' does not exist."
    echo "Please create the role first or ensure you have the correct role name."
    echo "You can use the role from the examples/lambda directory or create a new one with DynamoDB permissions."
    exit 1
fi

# Create DynamoDB policy for the rate limiter table
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
DYNAMODB_POLICY_NAME="ImposterRateLimiterDynamoDBPolicy"

echo "Creating/updating DynamoDB policy for rate limiter"
cat > /tmp/dynamodb-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "dynamodb:GetItem",
                "dynamodb:PutItem",
                "dynamodb:Query",
                "dynamodb:DeleteItem"
            ],
            "Resource": "arn:aws:dynamodb:${AWS_REGION:-us-east-1}:${ACCOUNT_ID}:table/${DYNAMODB_TABLE_NAME}"
        }
    ]
}
EOF

# Create or update the policy
aws iam create-policy \
    --policy-name "$DYNAMODB_POLICY_NAME" \
    --policy-document file:///tmp/dynamodb-policy.json \
    --description "DynamoDB permissions for Imposter rate limiter" 2>/dev/null || \
aws iam create-policy-version \
    --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${DYNAMODB_POLICY_NAME}" \
    --policy-document file:///tmp/dynamodb-policy.json \
    --set-as-default 2>/dev/null || echo "Policy update failed, continuing..."

# Attach the policy to the role
aws iam attach-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${DYNAMODB_POLICY_NAME}" 2>/dev/null || echo "Policy already attached"

# Clean up temporary file
rm -f /tmp/dynamodb-policy.json

echo "DynamoDB permissions configured for role '$ROLE_NAME'"

echo "=== Creating DynamoDB table for rate limiting ==="

# Check if DynamoDB table exists
echo "Checking if DynamoDB table '$DYNAMODB_TABLE_NAME' exists"
if ! aws dynamodb describe-table --table-name "$DYNAMODB_TABLE_NAME" > /dev/null 2>&1; then
    echo "Creating DynamoDB table '$DYNAMODB_TABLE_NAME'"
    aws dynamodb create-table \
        --table-name "$DYNAMODB_TABLE_NAME" \
        --attribute-definitions \
            AttributeName=StoreName,AttributeType=S \
            AttributeName=Key,AttributeType=S \
        --key-schema \
            AttributeName=StoreName,KeyType=HASH \
            AttributeName=Key,KeyType=RANGE \
        --billing-mode PAY_PER_REQUEST \
        --tags Key=Application,Value=ImposterRateLimiting Key=Purpose,Value=RateLimit
    
    echo "Waiting for table to become active..."
    aws dynamodb wait table-exists --table-name "$DYNAMODB_TABLE_NAME"
    echo "DynamoDB table '$DYNAMODB_TABLE_NAME' created successfully"
    
    echo "Enabling TTL on DynamoDB table"
    aws dynamodb update-time-to-live \
        --table-name "$DYNAMODB_TABLE_NAME" \
        --time-to-live-specification Enabled=true,AttributeName=ttl
    echo "TTL enabled on attribute 'ttl'"
else
    echo "DynamoDB table '$DYNAMODB_TABLE_NAME' already exists"
    
    # Check if TTL is enabled, enable if not
    echo "Checking TTL status on existing table"
    TTL_STATUS=$(aws dynamodb describe-time-to-live --table-name "$DYNAMODB_TABLE_NAME" --query 'TimeToLiveDescription.TimeToLiveStatus' --output text 2>/dev/null || echo "DISABLED")
    
    if [ "$TTL_STATUS" != "ENABLED" ]; then
        echo "Enabling TTL on existing DynamoDB table"
        aws dynamodb update-time-to-live \
            --table-name "$DYNAMODB_TABLE_NAME" \
            --time-to-live-specification Enabled=true,AttributeName=ttl
        echo "TTL enabled on attribute 'ttl'"
    else
        echo "TTL is already enabled on the table"
    fi
fi

echo "=== Building Lambda function ==="

echo "Building binary for Lambda"
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap ../../../cmd/imposter

echo "Creating config directory"
mkdir -p config
cp imposter-config.yaml config/

echo "Zipping binary and config"
zip -r imposter-rate-limiting.zip bootstrap config

echo "=== Deploying Lambda function ==="

# Set environment variables for DynamoDB store
LAMBDA_ENV_VARS="IMPOSTER_STORE_DRIVER=store-dynamodb,IMPOSTER_DYNAMODB_TABLE=$DYNAMODB_TABLE_NAME,IMPOSTER_DYNAMODB_REGION=${AWS_REGION:-us-east-1},IMPOSTER_DYNAMODB_TTL=300,IMPOSTER_DYNAMODB_TTL_ATTRIBUTE=ttl"

# Check if the Lambda function exists
echo "Checking if Lambda function exists"
if ! aws lambda get-function --function-name "$LAMBDA_FUNCTION_NAME" > /dev/null 2>&1; then
    echo "Creating Lambda function"
    aws lambda create-function --function-name "$LAMBDA_FUNCTION_NAME" \
        --runtime provided.al2023 \
        --handler bootstrap \
        --architectures x86_64 \
        --role "$LAMBDA_EXECUTION_ROLE" \
        --zip-file fileb://imposter-rate-limiting.zip \
        --environment Variables="{$LAMBDA_ENV_VARS}" \
        --timeout 30 \
        --memory-size 512 \
        --description "Imposter rate limiting example with DynamoDB backend"

else
    echo "Updating Lambda function code"
    aws lambda update-function-code --function-name "$LAMBDA_FUNCTION_NAME" \
        --zip-file fileb://imposter-rate-limiting.zip
    
    echo "Updating Lambda function environment variables"
    aws lambda update-function-configuration --function-name "$LAMBDA_FUNCTION_NAME" \
        --environment Variables="{$LAMBDA_ENV_VARS}"
fi

# Create function URL with public access
echo "Setting up Lambda function URL with public access"

echo "Getting Lambda function URL"
aws lambda get-function-url-config --function-name "$LAMBDA_FUNCTION_NAME" 2>/dev/null || \
  aws lambda create-function-url-config \
    --function-name "$LAMBDA_FUNCTION_NAME" \
    --auth-type NONE

echo "Adding public access permission for function URL"
aws lambda add-permission \
    --function-name "$LAMBDA_FUNCTION_NAME" \
    --statement-id FunctionURLAllowPublicAccess \
    --action lambda:InvokeFunctionUrl \
    --principal "*" \
    --function-url-auth-type NONE 2>/dev/null || echo "Permission already exists"

echo ""
echo "=== Deployment Complete ==="
echo "DynamoDB Table: $DYNAMODB_TABLE_NAME"
echo "Lambda Function: $LAMBDA_FUNCTION_NAME"
echo ""
echo "Lambda function URL:"
aws lambda get-function-url-config --function-name "$LAMBDA_FUNCTION_NAME" --query 'FunctionUrl' --output text

echo ""
echo "Environment variables configured:"
echo "  IMPOSTER_STORE_DRIVER=store-dynamodb"
echo "  IMPOSTER_DYNAMODB_TABLE=$DYNAMODB_TABLE_NAME"
echo "  IMPOSTER_DYNAMODB_REGION=${AWS_REGION:-us-east-1}"
echo "  IMPOSTER_DYNAMODB_TTL=300 (5 minutes)"
echo "  IMPOSTER_DYNAMODB_TTL_ATTRIBUTE=ttl"

echo ""
echo "Test the rate limiting endpoints:"
echo "  GET  \$FUNCTION_URL/api/light     (max 10 concurrent)"
echo "  GET  \$FUNCTION_URL/api/heavy     (tiered: 3->throttle, 5->503)"
echo "  POST \$FUNCTION_URL/api/critical  (max 2 concurrent)"
echo "  GET  \$FUNCTION_URL/api/database  (tiered: 5->throttle, 8->503)" 
echo "  POST \$FUNCTION_URL/api/upload    (max 1 concurrent)"
echo "  GET  \$FUNCTION_URL/api/status    (no rate limiting)"
echo "  GET  \$FUNCTION_URL/health        (no rate limiting)"

# Clean up temporary files
rm -f bootstrap imposter-rate-limiting.zip
rm -rf config