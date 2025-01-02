package soap

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
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
		WSDLFile: "petstore.wsdl",
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
					StatusCode: 200,
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir)
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

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle request
	responseState := handler.HandleRequest(req)
	responseState.WriteToResponseWriter(rr)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/soap+xml") {
		t.Errorf("Expected Content-Type to contain application/soap+xml, got %s", rr.Header().Get("Content-Type"))
	}
	if !strings.Contains(rr.Body.String(), "<getPetByIdResponse") {
		t.Errorf("Expected response to contain getPetByIdResponse, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "<n>Test Pet<n>") {
		t.Errorf("Expected response to contain Test Pet, got %s", rr.Body.String())
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
	handler, err := NewHandler(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}

	// Create test request with GET method
	req := httptest.NewRequest(http.MethodGet, "/pets/", nil)

	// Handle request
	responseState := handler.HandleRequest(req)

	// Check response
	if responseState.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, responseState.StatusCode)
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
	handler, err := NewHandler(cfg, ".")
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

	// Handle request
	responseState := handler.HandleRequest(req)

	// Check response
	if responseState.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, responseState.StatusCode)
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
	handler, err := NewHandler(cfg, ".")
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

	// Handle request
	responseState := handler.HandleRequest(req)

	// Check response - should be intercepted
	if responseState.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, responseState.StatusCode)
	}
	if !strings.Contains(responseState.Headers["Content-Type"], "application/soap+xml") {
		t.Errorf("Expected Content-Type to contain application/soap+xml, got %s", responseState.Headers["Content-Type"])
	}
	if !strings.Contains(string(responseState.Body), "<env:Fault>") {
		t.Errorf("Expected response to contain <env:Fault>, got %s", string(responseState.Body))
	}
	if !strings.Contains(string(responseState.Body), "<faultstring>Intercepted request</faultstring>") {
		t.Errorf("Expected response to contain intercepted message, got %s", string(responseState.Body))
	}
}

func TestSOAPHandler_HandleRequest_WithPassthroughInterceptor(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Copy test WSDL file to temp directory
	wsdlContent, err := os.ReadFile("testdata/petstore.wsdl")
	if err != nil {
		t.Fatal(err)
	}
	wsdlPath := filepath.Join(tempDir, "petstore.wsdl")
	err = os.WriteFile(wsdlPath, wsdlContent, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create test configuration with passthrough interceptor
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: wsdlPath,
		Interceptors: []config.Interceptor{
			{
				RequestMatcher: config.RequestMatcher{
					Method:     "POST",
					Path:       "/pets/",
					Operation:  "getPetById",
					SOAPAction: "getPetById",
				},
				Response: &config.Response{
					Content: "Intercepted but continuing",
					Headers: map[string]string{
						"X-Intercepted": "true",
					},
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
            <name>Test Pet</name>
        </getPetByIdResponse>
    </env:Body>
</env:Envelope>`,
					StatusCode: 200,
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir)
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

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle request
	responseState := handler.HandleRequest(req)
	responseState.WriteToResponseWriter(rr)

	// Check response - should be the final response, but with interceptor header
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/soap+xml") {
		t.Errorf("Expected Content-Type to contain application/soap+xml, got %s", rr.Header().Get("Content-Type"))
	}
	if rr.Header().Get("X-Intercepted") != "true" {
		t.Errorf("Expected X-Intercepted header to be true, got %s", rr.Header().Get("X-Intercepted"))
	}
	if !strings.Contains(rr.Body.String(), "<getPetByIdResponse") {
		t.Errorf("Expected response to contain getPetByIdResponse, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "<name>Test Pet</name>") {
		t.Errorf("Expected response to contain Test Pet, got %s", rr.Body.String())
	}
}
