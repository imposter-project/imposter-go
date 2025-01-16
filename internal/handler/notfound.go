package handler

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/config"
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
		case "openapi", "rest":
			for _, resource := range cfg.Resources {
				resInfo := describeResource(resource, "GET")
				restResources = append(restResources, resInfo)
			}
		case "soap":
			for _, resource := range cfg.Resources {
				var resInfo string
				if resource.Operation != "" || resource.Binding != "" {
					resInfo = fmt.Sprintf("Operation: %s (Binding: %s)", resource.Operation, resource.Binding)
				} else {
					resInfo = describeResource(resource, "POST")
				}
				soapResources = append(soapResources, resInfo)
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

// describeResource returns a string representation of a resource
func describeResource(resource config.Resource, defaultMethod string) string {
	if resource.Method == "" {
		resource.Method = defaultMethod
	}
	resInfo := fmt.Sprintf("%s %s", resource.Method, resource.Path)
	return resInfo
}
