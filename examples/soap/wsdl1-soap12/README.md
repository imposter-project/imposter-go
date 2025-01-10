# WSDL 1.1 with SOAP 1.2

This example demonstrates how to use WSDL 1.1 with SOAP 1.2 bindings.

## Running the example

```sh
imposter-go run examples/soap/wsdl1-soap12
```

## Testing the example using curl

```sh
curl http://localhost:8080/soap/ -H 'SOAPAction: getPetById' -H 'Content-Type: text/xml; charset=ISO-8859-1' -H 'Accept: text/xml' --data '<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
  <env:Header/> 
  <env:Body>
    <getPetByIdRequest xmlns="urn:com:example:petstore">
      <id>3</id>
    </getPetByIdRequest>
  </env:Body>
</env:Envelope>'
```
