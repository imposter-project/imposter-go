# SOAP Fault Example

This example demonstrates how to return SOAP fault responses using Imposter.

## Files

- `petstore.wsdl` - The WSDL file describing the pet store service
- `imposter-config.yaml` - The Imposter configuration file that returns a SOAP fault

## Running the Example

1. Start Imposter:
   ```bash
   imposter .
   ```

2. Send a SOAP request:
   ```bash
   curl -X POST "http://localhost:8080/pets/" \
       -H 'Content-Type: application/soap+xml' \
       -H 'SOAPAction: getPetById' \
       -d '<?xml version="1.0" encoding="UTF-8"?>
           <env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
               <env:Header/>
               <env:Body>
                   <getPetByIdRequest xmlns="urn:com:example:petstore">
                       <id>3</id>
                   </getPetByIdRequest>
               </env:Body>
           </env:Envelope>'
   ```

   Expected response (HTTP 500):
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
       <env:Header/>
       <env:Body>
           <env:Fault>
               <faultcode>env:Server</faultcode>
               <faultstring>Internal server error</faultstring>
               <detail>
                   <fault xmlns="urn:com:example:petstore">
                       <code>ERR-001</code>
                       <message>Failed to retrieve pet details</message>
                   </fault>
               </detail>
           </env:Fault>
       </env:Body>
   </env:Envelope>
   ```

## Configuration Details

The example demonstrates:

1. Returning a SOAP fault response with a 500 status code
2. Including fault details in the response
3. Using the fault schema defined in the WSDL 