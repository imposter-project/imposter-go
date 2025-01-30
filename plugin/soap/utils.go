package soap

import "fmt"

// Constants for SOAP content types
const (
	textXMLContentType = "text/xml"
	soap11ContentType  = textXMLContentType
	soap12ContentType  = "application/soap+xml"
)

// wrapInEnvelope wraps the given XML content in a SOAP envelope,
// using the specified envelope namespace.
// The prefix "env" is used for the envelope namespace.
func wrapInEnvelope(content string, envNamespace string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
  <env:Body>%s</env:Body>
</env:Envelope>`, envNamespace, content)
}

// getResponseContentType returns the content type for the SOAP response
// based on the SOAP version
func getResponseContentType(soapVersion SOAPVersion) string {
	switch {
	case soapVersion == SOAP12:
		return soap12ContentType
	default:
		return soap11ContentType
	}
}

// getResponseContentType returns a possible SOAP version for the given WSDL version
func guessSoapVersion(version WSDLVersion) SOAPVersion {
	switch version {
	case WSDL1:
		return SOAP11
	case WSDL2:
		return SOAP12
	default:
		return SOAP11
	}
}

// guessEnvNamespace guesses the envelope namespace based on the SOAP version
func guessEnvNamespace(soapVersion SOAPVersion) string {
	switch soapVersion {
	case SOAP11:
		return SOAP11EnvNamespace
	case SOAP12:
		return SOAP12RecEnvNamespace
	default:
		return ""
	}
}
