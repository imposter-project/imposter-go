# Test results of rate limiting on AWS Lambda using DynamoDB

## Test setup

- **API**: AWS Lambda function with DynamoDB for rate limiting
- **Test tool**: `hey`
- **Lambda configuration**:
  - Memory: 512 MB
  - Timeout: 30 seconds
  - Runtime: `provided.al2023`
- **DynamoDB configuration**:
  - Table: `imposter-rate-limiter`
  - Billing mode: Pay-per-request
  - TTL enabled on `ttl` attribute
- **Environment variables**:
  - `IMPOSTER_STORE_DRIVER=store-dynamodb`
  - `IMPOSTER_DYNAMODB_TTL=300` (5 minutes)
  - `IMPOSTER_DYNAMODB_TTL_ATTRIBUTE=ttl`

## Expected behavior

Based on the `/api/simple` endpoint configuration:

- **Concurrency limit**: 10 concurrent requests maximum
- **Normal response**: HTTP 200 with 1-second delay per request
- **Rate limited response**: HTTP 429 when >10 concurrent requests
- **Rate limit delay**: Additional 1-second delay when rate limited
- **Test scenario**: 20 concurrent clients for 10 seconds
  - Expected: ~50% of requests should get HTTP 429 (rate limited)
  - Expected: ~50% of requests should get HTTP 200 (successful)
  - All responses should have ~1-2 second response times (1s base + potential rate limit delay)

## Test results

```shell
$ hey -c 20 -z 10s https://endpoint-of-lambda-url.lambda-url.eu-west-2.on.aws/api/simple

Summary:
  Total:	10.1773 secs
  Slowest:	2.8958 secs
  Fastest:	1.0319 secs
  Average:	1.1362 secs
  Requests/sec:	17.5881

  Total data:	19841 bytes
  Size/request:	110 bytes

Response time histogram:
  1.032 [1]	|
  1.218 [158]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  1.405 [0]	|
  1.591 [0]	|
  1.777 [14]	|■■■■
  1.964 [5]	|■
  2.150 [0]	|
  2.337 [0]	|
  2.523 [0]	|
  2.709 [0]	|
  2.896 [1]	|


Latency distribution:
  10% in 1.0373 secs
  25% in 1.0405 secs
  50% in 1.0449 secs
  75% in 1.0766 secs
  90% in 1.7556 secs
  95% in 1.7721 secs
  99% in 2.8958 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0078 secs, 1.0319 secs, 2.8958 secs
  DNS-lookup:	0.0028 secs, 0.0000 secs, 0.0255 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0004 secs
  resp wait:	1.1282 secs, 1.0317 secs, 2.8231 secs
  resp read:	0.0001 secs, 0.0000 secs, 0.0031 secs

Status code distribution:
  [200]	90 responses
  [429]	89 responses
```
