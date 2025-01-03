# AWS Lambda Deployment Example

This directory contains the necessary files to build and deploy Imposter as an AWS Lambda function. The deployment uses the AWS Lambda custom runtime with Amazon Linux 2023.

## Prerequisites

- AWS CLI installed and configured
- AWS credentials with appropriate permissions to create/update Lambda functions
- AWS IAM role named `ImposterLambdaExecutionRole` with necessary permissions
- `AWS_REGION` environment variable set (unless region is set elsewhere)

## Directory Contents

- `build-deploy.sh`: Script to build and deploy the Lambda function
- `config/`: Configuration directory for Imposter
- `bootstrap`: Binary file (generated during build)
- `imposter.zip`: Deployment package (generated during build)

## Deployment Steps

1. Ensure your AWS credentials are configured:
   ```bash
   aws configure
   ```

2. Set your AWS region (optional):
   ```bash
   export AWS_REGION=your-preferred-region
   ```

3. Run the deployment script:
   ```bash
   ./build-deploy.sh
   ```

The script will:
- Build the binary for Linux/AMD64
- Package the binary and config into a zip file
- Create or update the Lambda function
- Configure a public function URL
- Display the function URL for accessing the service

## Function URL

After deployment, the Lambda function will be accessible via a function URL. This URL will be displayed at the end of the deployment process.

## Configuration

The `config/` directory contains the Imposter configuration files. Modify these files to customise the behaviour of your mock service.

## Security Note

The deployment script configures the Lambda function URL with public access (`auth-type NONE`). In a production environment, you may want to modify this to use IAM authentication or implement additional security measures. 