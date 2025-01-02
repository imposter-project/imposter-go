package soap

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/assert"
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
					Path: "/pets/",
				},
				Operation: &config.SOAPOperation{
					Name:       "getPetById",
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
	assert.NoError(t, err)

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
	handler.HandleRequest(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/soap+xml")
	assert.Contains(t, rr.Body.String(), "<getPetByIdResponse")
	assert.Contains(t, rr.Body.String(), "<n>Test Pet<n>")
}

func TestSOAPHandler_HandleRequest_InvalidMethod(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path: "/pets/",
				},
				Operation: &config.SOAPOperation{
					Name:       "getPetById",
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
	assert.NoError(t, err)

	// Create test request with GET method
	req := httptest.NewRequest(http.MethodGet, "/pets/", nil)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(rr, req)

	// Check response
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestSOAPHandler_HandleRequest_NoMatchingOperation(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "petstore.wsdl"),
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path: "/pets/",
				},
				Operation: &config.SOAPOperation{
					Name:       "getPetById",
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
	assert.NoError(t, err)

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

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(rr, req)

	// Check response
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestSOAPHandler_HandleRequest_WithInterceptor(t *testing.T) {
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

	// Create test configuration with interceptor
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: wsdlPath,
		Interceptors: []config.Interceptor{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "POST",
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
					StatusCode: 400,
				},
				Continue: false,
			},
		},
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Path: "/pets/",
				},
				Operation: &config.SOAPOperation{
					Name:       "getPetById",
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
	assert.NoError(t, err)

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

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(rr, req)

	// Check response - should be intercepted
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/soap+xml")
	assert.Contains(t, rr.Body.String(), "<env:Fault>")
	assert.Contains(t, rr.Body.String(), "<faultstring>Intercepted request</faultstring>")
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
					Method: "POST",
					Path:   "/pets/",
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
					Path: "/pets/",
				},
				Operation: &config.SOAPOperation{
					Name:       "getPetById",
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
	assert.NoError(t, err)

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
	handler.HandleRequest(rr, req)

	// Check response - should be the final response, but with interceptor header
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/soap+xml")
	assert.Equal(t, "true", rr.Header().Get("X-Intercepted"))
	assert.Contains(t, rr.Body.String(), "<getPetByIdResponse")
	assert.Contains(t, rr.Body.String(), "<name>Test Pet</name>")
}
