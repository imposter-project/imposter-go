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
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
    <env:Body>
        <getPetByIdResponse xmlns="urn:com:example:petstore">
            <id>3</id>
            <n>Test Pet<n>
        </getPetByIdResponse>
    </env:Body>
</env:Envelope>`,
					StatusCode: 200,
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope" xmlns:pet="urn:com:example:petstore">
    <env:Header/>
    <env:Body>
        <pet:getPetByIdRequest>
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
	if !strings.Contains(responseBody, "<getPetByIdResponse") {
		t.Errorf("Expected response to contain getPetByIdResponse, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "<n>Test Pet<n>") {
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
	handler, err := NewHandler(cfg, ".", &config.ImposterConfig{})
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
	handler, err := NewHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with different operation
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
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
					Headers: map[string]config.MatcherUnmarshaler{
						"X-Test-Header": {Matcher: config.StringMatcher("test-value")},
					},
				},
				Response: &config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
    <env:Body>
        <env:Fault>
            <faultcode>env:Client</faultcode>
            <faultstring>Intercepted request</faultstring>
        </env:Fault>
    </env:Body>
</env:Envelope>`,
					StatusCode: http.StatusBadRequest,
				},
				Continue: false,
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
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
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
	handler, err := NewHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
    <env:Body>
        <getPetByIdRequest xmlns="urn:com:example:petstore">
            <id>3</id>
        </getPetByIdRequest>
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
		t.Error("Expected response to be handled for interceptor")
	}

	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}

	responseBody := string(responseState.Body)
	if !strings.Contains(responseBody, "<env:Fault>") {
		t.Errorf("Expected response to contain Fault, got %s", responseBody)
	}
	if !strings.Contains(responseBody, "Intercepted request") {
		t.Errorf("Expected response to contain interceptor message, got %s", responseBody)
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
					Headers: map[string]config.MatcherUnmarshaler{
						"X-Test-Header": {Matcher: config.StringMatcher("test-value")},
					},
				},
				Response: &config.Response{
					Content: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
    <env:Body>
        <env:Fault>
            <faultcode>env:Client</faultcode>
            <faultstring>Intercepted request</faultstring>
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
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
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
	handler, err := NewHandler(cfg, ".", &config.ImposterConfig{})
	if err != nil {
		t.Fatal(err)
	}

	// Create test request
	soapRequest := `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2001/12/soap-envelope">
    <env:Header/>
    <env:Body>
        <getPetByIdRequest xmlns="urn:com:example:petstore">
            <id>3</id>
        </getPetByIdRequest>
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

	// Check response - should be the resource response since interceptor has Continue=true
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
