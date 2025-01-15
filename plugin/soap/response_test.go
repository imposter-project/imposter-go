package soap

import (
	"strings"
	"testing"
)

func TestWrapInEnvelope(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		soapVersion SOAPVersion
		want        string
	}{
		{
			name:        "SOAP 1.1 envelope with simple content",
			content:     "<test>Hello</test>",
			soapVersion: SOAP11,
			want: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://schemas.xmlsoap.org/soap/envelope/">
  <env:Body><test>Hello</test></env:Body>
</env:Envelope>`,
		},
		{
			name:        "SOAP 1.2 envelope with simple content",
			content:     "<test>Hello</test>",
			soapVersion: SOAP12,
			want: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
  <env:Body><test>Hello</test></env:Body>
</env:Envelope>`,
		},
		{
			name:        "Empty content",
			content:     "",
			soapVersion: SOAP11,
			want: `<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://schemas.xmlsoap.org/soap/envelope/">
  <env:Body></env:Body>
</env:Envelope>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapInEnvelope(tt.content, tt.soapVersion)
			// Normalise line endings for comparison
			got = strings.ReplaceAll(got, "\r\n", "\n")
			want := strings.ReplaceAll(tt.want, "\r\n", "\n")
			if got != want {
				t.Errorf("wrapInEnvelope() = %v, want %v", got, want)
			}
		})
	}
}
