# Expression Matching Example

This example demonstrates the use of the `allOf` matcher, which allows you to evaluate expressions and match their values against expected results.

## Overview

The `allOf` matcher evaluates expressions using Imposter's template syntax and compares the results using standard matching operators. This is useful for:
- Matching against store values
- Complex request matching using multiple conditions
- Combining different sources of data (query params, headers, store values, etc.)

## Example Configuration

The configuration shows several ways to use expression matching:

1. Simple store value matching:
```yaml
allOf:
  - expression: "${stores.example.foo}"
    value: "bar"
```
Matches when the value of `foo` in the `example` store equals "bar".

2. Using different operators:
```yaml
allOf:
  - expression: "${stores.example.baz}"
    operator: NotEqualTo
    value: "qux"
```
Matches when the value of `baz` in the `example` store is not equal to "qux".

3. Checking for existence:
```yaml
allOf:
  - expression: "${stores.example.exists}"
    operator: Exists
```
Matches when the `exists` key is present in the `example` store (value doesn't matter).

4. Multiple conditions:
```yaml
allOf:
  - expression: "${context.request.queryParams.foo}"
    value: "bar"
  - expression: "${context.request.queryParams.baz}"
    operator: Contains
    value: "qux"
```
Matches when:
- The query parameter `foo` equals "bar" AND
- The query parameter `baz` contains the string "qux"

## Available Operators

- `EqualTo` (default if not specified): Exact match
- `NotEqualTo`: Value must not match
- `Exists`: Value must be present
- `NotExists`: Value must not be present
- `Contains`: Value must contain the specified string
- `NotContains`: Value must not contain the specified string
- `Matches`: Value must match the specified regex pattern
- `NotMatches`: Value must not match the specified regex pattern

## Testing the Example

You can test these examples with curl:

```bash
# Test store value matching (assuming foo=bar in example store)
curl http://localhost:8080/example

# Test query parameter matching
curl "http://localhost:8080/example?foo=bar&baz=contains-qux-somewhere"
``` 