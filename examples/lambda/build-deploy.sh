#!/usr/bin/env bash
set -e

# Check if AWS_REGION is set
if [ -z "${AWS_REGION}" ]; then
    echo "Warning: AWS_REGION environment variable is not set. AWS commands may fail or use default region."
fi

echo "Building binary for Lambda"
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap ../../cmd/imposter                          

echo "Zipping binary and config"
zip -r imposter.zip bootstrap config                     

# check if the Lambda function exists
echo "Checking if Lambda function exists"
aws lambda get-function --function-name imposter-go > /dev/null 2>&1

if [ $? -ne 0 ]; then
    echo "Creating Lambda function"
    aws lambda create-function --function-name imposter-go \
        --runtime provided.al2023 \
        --handler bootstrap \
        --architectures x86_64 \
        --role ImposterLambdaExecutionRole \
        --zip-file fileb://imposter.zip

else
    echo "Updating Lambda function"
    aws lambda update-function-code --function-name imposter-go \
        --zip-file fileb://imposter.zip
fi

# Check if function URL is enabled and has public access
echo "Checking Lambda function URL permissions"
aws lambda get-function-url-config --function-name imposter-go > /dev/null 2>&1
if [ $? -eq 0 ]; then
    # Function URL exists, add public access if not already set
    aws lambda add-permission --function-name imposter-go \
        --statement-id FunctionURLAllowPublicAccess \
        --action lambda:InvokeFunctionUrl \
        --principal "*" \
        --function-url-auth-type NONE \
        --output text 2>/dev/null || true
fi

echo "Getting Lambda function URL"
aws lambda get-function-url-config --function-name imposter-go 2>/dev/null || \
  aws lambda create-function-url-config \
    --function-name imposter-go \
    --auth-type NONE

echo "Lambda function URL:"
aws lambda get-function-url-config --function-name imposter-go --query 'FunctionUrl' --output text

