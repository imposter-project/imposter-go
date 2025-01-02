# SOAP Interceptor Example

This example demonstrates how to use interceptors with SOAP requests in Imposter.

## Files

- `petstore.wsdl` - The WSDL file describing the pet store service
- `imposter-config.yaml` - The Imposter configuration file with interceptors

## Features Demonstrated

1. Request logging using capture
2. Authentication check using header matching
3. Template response using captured values
4. SOAP fault responses

## Running the Example

1. Start Imposter:
   ```bash
   imposter .
   ```

2. Try a request without authentication:
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

   Expected response (HTTP 401):
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
       <env:Header/>
       <env:Body>
           <env:Fault>
               <faultcode>env:Client</faultcode>
               <faultstring>Authentication required</faultstring>
           </env:Fault>
       </env:Body>
   </env:Envelope>
   ```

3. Try a request with authentication:
   ```bash
   curl -X POST "http://localhost:8080/pets/" \
       -H 'Content-Type: application/soap+xml' \
       -H 'SOAPAction: getPetById' \
       -H 'Authorization: Bearer test-token' \
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

   Expected response:
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
       <env:Header/>
       <env:Body>
           <getPetByIdResponse xmlns="urn:com:example:petstore">
               <id>3</id>
               <name>Pet 3</name>
           </getPetByIdResponse>
       </env:Body>
   </env:Envelope>
   ```

## Configuration Details

The example demonstrates:

1. Using interceptors to validate requests before they reach the main handler
2. Capturing request data for use in responses
3. Using templates to include captured data in responses
4. Returning SOAP faults for authentication failures 