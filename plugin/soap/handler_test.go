package soap

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/response"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

func TestSOAPHandler_HandleRequest(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Copy test WSDL file to temp directory
	wsdlContent, err := os.ReadFile("testdata/petstore.wsdl")
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "petstore.wsdl"), wsdlContent, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join(tempDir, "petstore.wsdl"),
		System: &config.System{
			XMLNamespaces: map[string]string{
				"pet": "urn:com:example:petstore",
			},
		},
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path:       "/pets/",
					Operation:  "getPetById",
					SOAPAction: "getPetById",
					RequestBody: config.RequestBody{
						BodyMatchCondition: config.BodyMatchCondition{
							MatchCondition: config.MatchCondition{
								Value: "3",
							},
							XPath: "//pet:id",
						},
					},
				},
				Response: config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
    <env:Header/>
    <env:Body>
        <pet:getPetByIdResponse xmlns:pet="urn:com:example:petstore">
            <pet:id>3</pet:id>
            <pet:name>Test Pet</pet:name>
        </pet:getPetByIdResponse>
    </env:Body>
</env:Envelope>`,
					StatusCode: 200,
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
    <env:Header/>
    <env:Body>
        <pet:getPetByIdRequest xmlns:pet="urn:com:example:petstore">
            <pet:id>3</pet:id>
        </pet:getPetByIdRequest>
    </env:Body>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("SOAPAction", "getPetById")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	if !strings.Contains(responseState.Headers["Content-Type"], "application/soap+xml") {
		t.Errorf("Expected Content-Type to contain application/soap+xml, got %s", responseState.Headers["Content-Type"])
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<pet:getPetByIdResponse") {
		t.Errorf("Expected response to contain getPetByIdResponse, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "<pet:name>Test Pet</pet:name>") {
		t.Errorf("Expected response to contain Test Pet, got %s", responseBody)
	}
}

func TestSOAPHandler_HandleRequest_InvalidMethod(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path:       "/pets/",
					Operation:  "getPetById",
					SOAPAction: "getPetById",
				},
				Response: config.Response{
					Content:    "test response",
					StatusCode: 200,
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with GET method
	req := httptest.NewRequest(http.MethodGet, "/pets/", nil)

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response - should not be handled by SOAP handler
	if responseState.Handled {
		t.Error("Expected response to not be handled for invalid method")
	}
}

func TestSOAPHandler_HandleRequest_NoMatchingOperation(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path:       "/pets/",
					Operation:  "getPetById",
					SOAPAction: "getPetById",
				},
				Response: config.Response{
					Content:    "test response",
					StatusCode: 200,
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with different operation
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
    <env:Body>
        <unknownOperation xmlns="urn:com:example:petstore">
            <id>3</id>
        </unknownOperation>
    </env:Body>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("SOAPAction", "unknownOperation")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if responseState.Handled {
		t.Error("Expected response to not be handled for no matching operation")
	}
}

func TestSOAPHandler_HandleRequest_WithInterceptor(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
		Interceptors: []config.Interceptor{
			{
				RequestMatcher: config.RequestMatcher{
					Method: http.MethodPost,
					Path:   "/pets/",
					RequestHeaders: map[string]config.MatcherUnmarshaler{
						"X-Test-Header": {Matcher: config.StringMatcher("test-value")},
					},
				},
				Response: &config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
   <env:Body>
      <env:Fault>
         <env:Code>
            <env:Value>env:Sender</env:Value>
         </env:Code>
         <env:Reason>
            <env:Text>Intercepted request</env:Text>
         </env:Reason>
      </env:Fault>
   </env:Body>
</env:Envelope>`,
					StatusCode: http.StatusBadRequest,
				},
				Continue: false,
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
   <env:Body>
      <getPetById xmlns="urn:com:example:petstore">
         <id>3</id>
      </getPetById>
   </env:Body>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("SOAPAction", "getPetById")
	req.Header.Set("X-Test-Header", "test-value")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<env:Fault>") {
		t.Errorf("Expected response to contain SOAP fault, got %s", responseBody)
	}
}

func TestSOAPHandler_HandleRequest_WithPassthroughInterceptor(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
		Interceptors: []config.Interceptor{
			{
				RequestMatcher: config.RequestMatcher{
					Method: http.MethodPost,
					Path:   "/pets/",
					RequestHeaders: map[string]config.MatcherUnmarshaler{
						"X-Test-Header": {Matcher: config.StringMatcher("test-value")},
					},
				},
				Response: &config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
   <env:Body>
      <env:Fault>
         <env:Code>
            <env:Value>env:Sender</env:Value>
         </env:Code>
         <env:Reason>
            <env:Text>Intercepted request</env:Text>
         </env:Reason>
      </env:Fault>
   </env:Body>
</env:Envelope>`,
					StatusCode: http.StatusBadRequest,
				},
				Continue: true,
			},
		},
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path:       "/pets/",
					Operation:  "getPetById",
					SOAPAction: "getPetById",
				},
				Response: config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
    <env:Body>
        <getPetByIdResponse xmlns="urn:com:example:petstore">
            <id>3</id>
            <n>Test Pet<n>
        </getPetByIdResponse>
    </env:Body>
</env:Envelope>`,
					StatusCode: http.StatusOK,
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
   <env:Body>
      <getPetById xmlns="urn:com:example:petstore">
         <id>3</id>
      </getPetById>
   </env:Body>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("SOAPAction", "getPetById")
	req.Header.Set("X-Test-Header", "test-value")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<getPetByIdResponse") {
		t.Errorf("Expected response to contain getPetByIdResponse, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "<n>Test Pet<n>") {
		t.Errorf("Expected response to contain Test Pet, got %s", responseBody)
	}
}

func TestSOAPHandler_HandleRequest_InvalidXML(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with invalid XML
	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
    <env:Body>
        <invalidTag>
            <unclosed>
    </env:Body>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(invalidXML))
	req.Header.Set("Content-Type", "application/soap+xml")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<env:Fault") {
		t.Errorf("Expected response to contain SOAP fault, got %s", responseBody)
	}
}

func TestSOAPHandler_HandleRequest_MissingBody(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with missing Body element
	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(invalidXML))
	req.Header.Set("Content-Type", "application/soap+xml")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<env:Fault") {
		t.Errorf("Expected response to contain SOAP fault, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "Invalid SOAP envelope") {
		t.Errorf("Expected fault to mention invalid SOAP envelope, got %s", responseBody)
	}
}

func TestSOAPHandler_SOAP11Fault(t *testing.T) {
	// Create test configuration with SOAP 1.1 WSDL
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore11.wsdl"), // This should be a SOAP 1.1 WSDL
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path:       "/pets/",
					Operation:  "getPetById",
					SOAPAction: "getPetById",
				},
				Response: config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
   <soapenv:Body>
      <soapenv:Fault>
         <faultcode>soapenv:Client</faultcode>
         <faultstring>Invalid pet ID</faultstring>
         <detail>
            <pet:getPetByIdFault xmlns:pet="urn:com:example:petstore">
                <pet:message>Pet ID must be a positive integer</pet:message>
            </pet:getPetByIdFault>
         </detail>
      </soapenv:Fault>
   </soapenv:Body>
</soapenv:Envelope>`,
					StatusCode: http.StatusBadRequest,
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
   <soapenv:Body>
      <pet:getPetByIdRequest xmlns:pet="urn:com:example:petstore">
         <pet:id>invalid</pet:id>
      </pet:getPetByIdRequest>
   </soapenv:Body>
</soapenv:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("SOAPAction", "getPetById")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<soapenv:Fault>") {
		t.Errorf("Expected response to contain SOAP 1.1 fault, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "<faultstring>Invalid pet ID</faultstring>") {
		t.Errorf("Expected fault to contain error message, got %s", responseBody)
	}
}

func TestSOAPHandler_SOAP12Fault(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path:       "/pets/",
					Operation:  "getPetById",
					SOAPAction: "getPetById",
				},
				Response: config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
   <env:Body>
      <env:Fault>
         <env:Code>
            <env:Value>env:Sender</env:Value>
            <env:Subcode>
               <env:Value>pet:InvalidId</env:Value>
            </env:Subcode>
         </env:Code>
         <env:Reason>
            <env:Text xml:lang="en">Invalid pet ID</env:Text>
         </env:Reason>
         <env:Detail>
            <pet:getPetByIdFault xmlns:pet="urn:com:example:petstore">
                <pet:message>Pet ID must be a positive integer</pet:message>
            </pet:getPetByIdFault>
         </env:Detail>
      </env:Fault>
   </env:Body>
</env:Envelope>`,
					StatusCode: http.StatusBadRequest,
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
   <env:Body>
      <pet:getPetByIdRequest xmlns:pet="urn:com:example:petstore">
         <pet:id>invalid</pet:id>
      </pet:getPetByIdRequest>
   </env:Body>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("SOAPAction", "getPetById")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<env:Fault>") {
		t.Errorf("Expected response to contain SOAP 1.2 fault, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "<env:Value>env:Sender</env:Value>") {
		t.Errorf("Expected fault to contain error code, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "<env:Text xml:lang=\"en\">Invalid pet ID</env:Text>") {
		t.Errorf("Expected fault to contain error message, got %s", responseBody)
	}
}

func TestSOAPHandler_InvalidSOAPVersion(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with invalid SOAP version namespace
	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://invalid.soap.version">
    <env:Body>
        <getPetById xmlns="urn:com:example:petstore">
            <id>1</id>
        </getPetById>
    </env:Body>
</env:Envelope>`

	req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(invalidXML))
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("SOAPAction", "getPetById")

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled")
	}

	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<env:Fault") {
		t.Errorf("Expected response to contain SOAP fault, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "Invalid SOAP envelope") {
		t.Errorf("Expected fault to mention invalid SOAP envelope, got %s", responseBody)
	}
}
