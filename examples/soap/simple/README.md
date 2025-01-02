# SOAP Example

This example demonstrates how to use Imposter to mock a SOAP web service.

## Files

- `petstore.wsdl` - The WSDL file describing the pet store service
- `imposter-config.yaml` - The Imposter configuration file

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

   Expected response:
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
       <env:Header/>
       <env:Body>
           <getPetByIdResponse xmlns="urn:com:example:petstore">
               <id>3</id>
               <name>Custom pet name</name>
           </getPetByIdResponse>
       </env:Body>
   </env:Envelope>
   ```

3. Try an invalid request:
   ```bash
   curl -X POST "http://localhost:8080/pets/" \
       -H 'Content-Type: application/soap+xml' \
       -H 'SOAPAction: invalid-pet-action' \
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

   Expected response (HTTP 400):
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
       <env:Header/>
       <env:Body>
           <env:Fault>
               <faultcode>env:Client</faultcode>
               <faultstring>Invalid SOAPAction</faultstring>
           </env:Fault>
       </env:Body>
   </env:Envelope>
   ```

## Configuration Details

The example demonstrates:

1. Basic SOAP request/response matching based on operation name and SOAPAction
2. Custom response bodies
3. Error handling with SOAP faults
4. WSDL-based service definition 