#!/usr/bin/env bash
set -e

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
