# SOAP Fault Example

This example demonstrates how to configure SOAP fault responses in Imposter, including:
- SOAP 1.1 and SOAP 1.2 fault messages
- Different fault scenarios
- Custom fault codes and messages
- HTTP status codes with faults

## Configuration

The example uses a WSDL file (`petstore.wsdl`) that defines a simple pet store service, and a configuration file (`imposter-config.yaml`) that sets up fault responses:

```yaml
plugin: soap
wsdlFile: petstore.wsdl
resources:
  - path: /pets/
    operation:
      name: getPetById
      soapAction: getPetById
    requestBody:
      xPath: "//id"
      value: "999"
    response:
      content: |
        <env:Fault xmlns:env="http://schemas.xmlsoap.org/soap/envelope/">
          <faultcode>env:Client</faultcode>
          <faultstring>Pet not found</faultstring>
          <detail>
            <error xmlns="urn:com:example:petstore">
              <code>NOT_FOUND</code>
              <message>Pet with ID 999 does not exist</message>
            </error>
          </detail>
        </env:Fault>
      statusCode: 404
      headers:
        Content-Type: application/soap+xml

  - path: /pets/
    operation:
      name: getPetById
      soapAction: getPetById
    requestBody:
      xPath: "//id"
      value: "0"
    response:
      content: |
        <env:Fault xmlns:env="http://www.w3.org/2003/05/soap-envelope">
          <env:Code>
            <env:Value>env:Sender</env:Value>
          </env:Code>
          <env:Reason>
            <env:Text>Invalid pet ID</env:Text>
          </env:Reason>
          <env:Detail>
            <error xmlns="urn:com:example:petstore">
              <code>INVALID_ID</code>
              <message>Pet ID must be a positive integer</message>
            </error>
          </env:Detail>
        </env:Fault>
      statusCode: 400
      headers:
        Content-Type: application/soap+xml
```

## Testing the Example

You can test the example using curl commands:

1. Request a non-existent pet (SOAP 1.1 fault):
```bash
curl -X POST -H "Content-Type: application/soap+xml" -H "SOAPAction: getPetById" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://schemas.xmlsoap.org/soap/envelope/">
    <env:Body>
        <getPetByIdRequest xmlns="urn:com:example:petstore">
            <id>999</id>
        </getPetByIdRequest>
    </env:Body>
</env:Envelope>' \
  http://localhost:8080/pets/
```
Expected response (404 Not Found):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://schemas.xmlsoap.org/soap/envelope/">
    <env:Body>
        <env:Fault>
            <faultcode>env:Client</faultcode>
            <faultstring>Pet not found</faultstring>
            <detail>
                <error xmlns="urn:com:example:petstore">
                    <code>NOT_FOUND</code>
                    <message>Pet with ID 999 does not exist</message>
                </error>
            </detail>
        </env:Fault>
    </env:Body>
</env:Envelope>
```

2. Request with invalid ID (SOAP 1.2 fault):
```bash
curl -X POST -H "Content-Type: application/soap+xml" -H "SOAPAction: getPetById" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
    <env:Body>
        <getPetByIdRequest xmlns="urn:com:example:petstore">
            <id>0</id>
        </getPetByIdRequest>
    </env:Body>
</env:Envelope>' \
  http://localhost:8080/pets/
```
Expected response (400 Bad Request):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
    <env:Body>
        <env:Fault>
            <env:Code>
                <env:Value>env:Sender</env:Value>
            </env:Code>
            <env:Reason>
                <env:Text>Invalid pet ID</env:Text>
            </env:Reason>
            <env:Detail>
                <error xmlns="urn:com:example:petstore">
                    <code>INVALID_ID</code>
                    <message>Pet ID must be a positive integer</message>
                </error>
            </env:Detail>
        </env:Fault>
    </env:Body>
</env:Envelope>
```

## Features Demonstrated

1. **SOAP Versions**: Support for both SOAP 1.1 and SOAP 1.2 fault messages with correct namespaces and structures.

2. **Fault Types**:
   - Client/Sender faults (invalid input)
   - Server/Receiver faults (internal errors)
   - Custom fault details

3. **HTTP Integration**:
   - Appropriate HTTP status codes (400, 404)
   - Correct Content-Type headers
   - SOAPAction header handling

4. **Request Matching**:
   - XPath-based request body matching
   - Operation name and SOAPAction matching
   - Path matching

5. **Error Details**:
   - Custom error codes
   - Descriptive error messages
   - Structured fault detail elements 