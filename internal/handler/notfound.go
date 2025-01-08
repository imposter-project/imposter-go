package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/plugin"
)

// handleNotFound generates a custom 404 page with available resources
func handleNotFound(r *http.Request, responseState *response.ResponseState, plugins []plugin.Plugin) {
	responseState.StatusCode = http.StatusNotFound
	responseState.Headers["Content-Type"] = "text/html"

	// Build list of available resources
	var restResources []string
	var soapResources []string

	for _, plg := range plugins {
		cfg := plg.GetConfig()
		switch cfg.Plugin {
		case "rest":
			for _, resource := range cfg.Resources {
				if resource.Method == "" {
					resource.Method = "GET" // Default to GET if not specified
				}
				restResources = append(restResources, fmt.Sprintf("%s %s", resource.Method, resource.Path))
			}
		case "soap":
			for _, resource := range cfg.Resources {
				soapResources = append(soapResources, fmt.Sprintf("Operation: %s (Path: %s)", resource.Operation, resource.Path))
			}
		}
	}

	// Build HTML response
	var html strings.Builder
	html.WriteString(`<html>
<head><title>Not found</title></head>
<body>
<h3>Resource not found</h3>
<p>
No resource exists for: <pre>`)
	html.WriteString(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
	html.WriteString("</pre></p>")

	// Add REST resources section if any exist
	if len(restResources) > 0 {
		html.WriteString("<p>The available REST resources are:\n<ul>")
		for _, resource := range restResources {
			html.WriteString(fmt.Sprintf("<li>%s</li>", resource))
		}
		html.WriteString("</ul></p>")
	}

	// Add SOAP resources section if any exist
	if len(soapResources) > 0 {
		html.WriteString("<p>The available SOAP operations are:\n<ul>")
		for _, resource := range soapResources {
			html.WriteString(fmt.Sprintf("<li>%s</li>", resource))
		}
		html.WriteString("</ul></p>")
	}

	html.WriteString(`<hr/>
<p>
<em><a href="https://www.imposter.sh">Imposter mock engine</a></em>
</p>
</body>
</html>`)

	responseState.Body = []byte(html.String())
}
