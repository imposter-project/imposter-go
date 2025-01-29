package soap

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/query"
	"github.com/imposter-project/imposter-go/internal/response"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

func TestSOAPHandler_HandleRequest(t *testing.T) {
	tests := []struct {
		name            string
		wsdlPath        string
		envelopeNS      string
		contentType     string
		responseContent string
		tnsPrefix       string
		xpathQueries    []string
	}{
		{
			name:        "WSDL 2.0 SOAP 1.2",
			wsdlPath:    "testdata/wsdl2-soap12/service.wsdl",
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
			responseContent: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
    <env:Header/>
    <env:Body>
        <pet:getPetByIdResponse xmlns:pet="urn:com:example:petstore">
            <pet:id>3</pet:id>
            <pet:name>Test Pet</pet:name>
        </pet:getPetByIdResponse>
    </env:Body>
</env:Envelope>`,
			tnsPrefix: "pet",
			xpathQueries: []string{
				"//pet:getPetByIdResponse[pet:id/text()='3']",
				"//pet:getPetByIdResponse[pet:name/text()='Test Pet']",
			},
		},
		{
			name:        "WSDL 1.1 SOAP 1.2",
			wsdlPath:    "testdata/wsdl1-soap12/service.wsdl",
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
			responseContent: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
    <env:Header/>
    <env:Body>
        <pet:getPetByIdResponse xmlns:pet="urn:com:example:petstore">
            <pet:id>3</pet:id>
            <pet:name>Test Pet</pet:name>
        </pet:getPetByIdResponse>
    </env:Body>
</env:Envelope>`,
			tnsPrefix: "pet",
			xpathQueries: []string{
				"//pet:getPetByIdResponse[pet:id/text()='3']",
				"//pet:getPetByIdResponse[pet:name/text()='Test Pet']",
			},
		},
		{
			name:        "WSDL 1.1 SOAP 1.1",
			wsdlPath:    "testdata/wsdl1-soap11/service.wsdl",
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
			responseContent: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
    <env:Header/>
    <env:Body>
        <pet:getPetByIdResponse xmlns:pet="urn:com:example:petstore">
            <pet:id>3</pet:id>
            <pet:name>Test Pet</pet:name>
        </pet:getPetByIdResponse>
    </env:Body>
</env:Envelope>`,
			tnsPrefix: "pet",
			xpathQueries: []string{
				"//pet:getPetByIdResponse[pet:id/text()='3']",
				"//pet:getPetByIdResponse[pet:name/text()='Test Pet']",
			},
		},
		{
			name:        "WSDL 1.1 SOAP 1.1 with Message Part Filter",
			wsdlPath:    "testdata/wsdl1-soap11-filter-message-parts/service.wsdl",
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
			// expect a response to be generated from the WSDL schema
			responseContent: "",
			tnsPrefix:       "pet",
			xpathQueries: []string{
				"//pet:getPetByIdResponse",
			},
		},
		{
			name:        "WSDL 1.1 SOAP 1.1 with Composite Message",
			wsdlPath:    "testdata/wsdl1-soap11-composite-message/service.wsdl",
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
			// expect a response to be generated from the WSDL schema
			responseContent: "",
			tnsPrefix:       "tns",
			// note: the id and name values are randomly generated
			xpathQueries: []string{
				"//tns:getPetByIdResponse/tns:id",
				"//tns:getPetByIdResponse/tns:name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test files
			tempDir := t.TempDir()

			// Copy test WSDL file to temp directory
			wsdlContent, err := os.ReadFile(tt.wsdlPath)
			if err != nil {
				t.Fatal(err)
			}
			err = os.WriteFile(filepath.Join(tempDir, "service.wsdl"), wsdlContent, 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Create test configuration
			cfg := &config.Config{
				Plugin:   "soap",
				WSDLFile: filepath.Join(tempDir, "service.wsdl"),
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
								BodyMatchCondition: &config.BodyMatchCondition{
									MatchCondition: config.MatchCondition{
										Value: "3",
									},
									XPath: "//pet:id",
								},
							},
						},
						Response: config.Response{
							StatusCode: 200,
						},
					},
				},
			}

			if tt.responseContent != "" {
				cfg.Resources[0].Response.Content = fmt.Sprintf(tt.responseContent, tt.envelopeNS)
			}

			// Create handler
			handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
			if err != nil {
				t.Fatal(err)
			}

			// Create test request
			soapRequest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
    <env:Header/>
    <env:Body>
        <pet:getPetByIdRequest xmlns:pet="urn:com:example:petstore">
            <pet:id>3</pet:id>
        </pet:getPetByIdRequest>
    </env:Body>
</env:Envelope>`, tt.envelopeNS)

			req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
			req.Header.Set("Content-Type", tt.contentType)
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

			if !strings.Contains(responseState.Headers["Content-Type"], tt.contentType) {
				t.Errorf("Expected Content-Type to contain %s, got %s", tt.contentType, responseState.Headers["Content-Type"])
			}

			responseBody := string(responseState.Body)
			ns := map[string]string{"env": tt.envelopeNS, tt.tnsPrefix: "urn:com:example:petstore"}

			// Execute XPath queries
			for _, xpathQuery := range tt.xpathQueries {
				result, success := query.XPathQuery(responseState.Body, xpathQuery, ns)
				if !success {
					t.Errorf("Failed to execute XPath query %s: %s", xpathQuery, responseBody)
					continue
				}
				if result == "" {
					t.Errorf("XPath query %s did not find a matching node in: %s", xpathQuery, responseBody)
				}
			}
		})
	}
}

func TestSOAPHandler_HandleRequest_InvalidMethod(t *testing.T) {
	tests := []struct {
		name     string
		wsdlPath string
	}{
		{
			name:     "WSDL 2.0 SOAP 1.2",
			wsdlPath: filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.2",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap12/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.1",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap11/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.1 with Message Part Filter",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap11-filter-message-parts/service.wsdl"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := &config.Config{
				Plugin:   "soap",
				WSDLFile: tt.wsdlPath,
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
		})
	}
}

func TestSOAPHandler_HandleRequest_NoMatchingOperation(t *testing.T) {
	tests := []struct {
		name        string
		wsdlPath    string
		envelopeNS  string
		contentType string
	}{
		{
			name:        "WSDL 2.0 SOAP 1.2",
			wsdlPath:    filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
		},
		{
			name:        "WSDL 1.1 SOAP 1.2",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap12/service.wsdl"),
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
		},
		{
			name:        "WSDL 1.1 SOAP 1.1",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap11/service.wsdl"),
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
		},
		{
			name:        "WSDL 1.1 SOAP 1.1 with Message Part Filter",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap11-filter-message-parts/service.wsdl"),
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := &config.Config{
				Plugin:   "soap",
				WSDLFile: tt.wsdlPath,
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
			soapRequest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
    <env:Body>
        <unknownOperation xmlns="urn:com:example:petstore">
            <id>3</id>
        </unknownOperation>
    </env:Body>
</env:Envelope>`, tt.envelopeNS)

			req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
			req.Header.Set("Content-Type", tt.contentType)
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
		})
	}
}

func TestSOAPHandler_HandleRequest_WithInterceptor(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Plugin:   "soap",
		WSDLFile: filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
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
		WSDLFile: filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
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
            <name>Test Pet</name>
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
	if !strings.Contains(responseBody, "<name>Test Pet</name>") {
		t.Errorf("Expected response to contain Test Pet, got %s", responseBody)
	}
}

func TestSOAPHandler_HandleRequest_InvalidXML(t *testing.T) {
	tests := []struct {
		name     string
		wsdlPath string
	}{
		{
			name:     "WSDL 2.0 SOAP 1.2",
			wsdlPath: filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.2",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap12/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.1",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap11/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.1 with Message Part Filter",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap11-filter-message-parts/service.wsdl"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := &config.Config{
				Plugin:   "soap",
				WSDLFile: tt.wsdlPath,
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
		})
	}
}

func TestSOAPHandler_HandleRequest_MissingBody(t *testing.T) {
	tests := []struct {
		name     string
		wsdlPath string
	}{
		{
			name:     "WSDL 2.0 SOAP 1.2",
			wsdlPath: filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.2",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap12/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.1",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap11/service.wsdl"),
		},
		{
			name:     "WSDL 1.1 SOAP 1.1 with Message Part Filter",
			wsdlPath: filepath.Join("testdata", "wsdl1-soap11-filter-message-parts/service.wsdl"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := &config.Config{
				Plugin:   "soap",
				WSDLFile: tt.wsdlPath,
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
		})
	}
}

func TestSOAPHandler_SOAPFault(t *testing.T) {
	tests := []struct {
		name            string
		wsdlPath        string
		envelopeNS      string
		contentType     string
		responseContent string
		expectedFault   string
		expectedReason  string
	}{
		{
			name:        "WSDL 2.0 SOAP 1.2",
			wsdlPath:    filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
			responseContent: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
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
			expectedFault:  "<env:Value>env:Sender</env:Value>",
			expectedReason: "<env:Text xml:lang=\"en\">Invalid pet ID</env:Text>",
		},
		{
			name:        "WSDL 1.1 SOAP 1.2",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap12/service.wsdl"),
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
			responseContent: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
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
			expectedFault:  "<env:Value>env:Sender</env:Value>",
			expectedReason: "<env:Text xml:lang=\"en\">Invalid pet ID</env:Text>",
		},
		{
			name:        "WSDL 1.1 SOAP 1.1",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap11/service.wsdl"),
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
			responseContent: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
   <env:Body>
      <env:Fault>
         <faultcode>env:Client</faultcode>
         <faultstring>Invalid pet ID</faultstring>
         <detail>
            <pet:getPetByIdFault xmlns:pet="urn:com:example:petstore">
                <pet:message>Pet ID must be a positive integer</pet:message>
            </pet:getPetByIdFault>
         </detail>
      </env:Fault>
   </env:Body>
</env:Envelope>`,
			expectedFault:  "<faultcode>env:Client</faultcode>",
			expectedReason: "<faultstring>Invalid pet ID</faultstring>",
		},
		{
			name:        "WSDL 1.1 SOAP 1.1 with Message Part Filter",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap11-filter-message-parts/service.wsdl"),
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
			responseContent: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
   <env:Body>
      <env:Fault>
         <faultcode>env:Client</faultcode>
         <faultstring>Invalid pet ID</faultstring>
         <detail>
            <pet:getPetByIdFault xmlns:pet="urn:com:example:petstore">
                <pet:message>Pet ID must be a positive integer</pet:message>
            </pet:getPetByIdFault>
         </detail>
      </env:Fault>
   </env:Body>
</env:Envelope>`,
			expectedFault:  "<faultcode>env:Client</faultcode>",
			expectedReason: "<faultstring>Invalid pet ID</faultstring>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := &config.Config{
				Plugin:   "soap",
				WSDLFile: tt.wsdlPath,
				Resources: []config.Resource{
					{
						RequestMatcher: config.RequestMatcher{
							Path:       "/pets/",
							Operation:  "getPetById",
							SOAPAction: "getPetById",
						},
						Response: config.Response{
							Content:    fmt.Sprintf(tt.responseContent, tt.envelopeNS),
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
			soapRequest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
   <env:Body>
      <pet:getPetByIdRequest xmlns:pet="urn:com:example:petstore">
         <pet:id>invalid</pet:id>
      </pet:getPetByIdRequest>
   </env:Body>
</env:Envelope>`, tt.envelopeNS)

			req := httptest.NewRequest(http.MethodPost, "/pets/", strings.NewReader(soapRequest))
			req.Header.Set("Content-Type", tt.contentType)
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
				t.Errorf("Expected response to contain SOAP fault, got %s", responseBody)
			}
			if !strings.Contains(responseBody, tt.expectedFault) {
				t.Errorf("Expected fault to contain error code %s, got %s", tt.expectedFault, responseBody)
			}
			if !strings.Contains(responseBody, tt.expectedReason) {
				t.Errorf("Expected fault to contain error message %s, got %s", tt.expectedReason, responseBody)
			}
		})
	}
}

func TestSOAPHandler_InvalidSOAPVersion(t *testing.T) {
	tests := []struct {
		name        string
		wsdlPath    string
		envelopeNS  string
		contentType string
	}{
		{
			name:        "WSDL 2.0 SOAP 1.2",
			wsdlPath:    filepath.Join("testdata", "wsdl2-soap12/service.wsdl"),
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
		},
		{
			name:        "WSDL 1.1 SOAP 1.2",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap12/service.wsdl"),
			envelopeNS:  "http://www.w3.org/2003/05/soap-envelope",
			contentType: "application/soap+xml",
		},
		{
			name:        "WSDL 1.1 SOAP 1.1",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap11/service.wsdl"),
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
		},
		{
			name:        "WSDL 1.1 SOAP 1.1 with Message Part Filter",
			wsdlPath:    filepath.Join("testdata", "wsdl1-soap11-filter-message-parts/service.wsdl"),
			envelopeNS:  "http://schemas.xmlsoap.org/soap/envelope/",
			contentType: "text/xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := &config.Config{
				Plugin:   "soap",
				WSDLFile: tt.wsdlPath,
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
			req.Header.Set("Content-Type", tt.contentType)
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
		})
	}
}
