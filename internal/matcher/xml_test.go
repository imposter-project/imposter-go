package matcher

import (
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMatchXPath(t *testing.T) {
	tests := []struct {
		name             string
		body             []byte
		condition        config.BodyMatchCondition
		systemNamespaces map[string]string
		want             bool
	}{
		{
			name: "simple element match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user>Grace</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user",
			},
			want: true,
		},
		{
			name: "nested element match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <users>
        <user>
            <name>Grace</name>
            <age>30</age>
        </user>
    </users>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user/name",
			},
			want: true,
		},
		{
			name: "attribute match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user id="123">Grace</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "123",
				},
				XPath: "//user/@id",
			},
			want: true,
		},
		{
			name: "with namespace",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root xmlns:ns="http://example.com">
    <ns:user>Grace</ns:user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//ns:user",
				XMLNamespaces: map[string]string{
					"ns": "http://example.com",
				},
			},
			want: true,
		},
		{
			name: "with system namespace",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root xmlns:sys="http://example.com/system">
    <sys:user>Grace</sys:user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//sys:user",
			},
			systemNamespaces: map[string]string{
				"sys": "http://example.com/system",
			},
			want: true,
		},
		{
			name: "system namespace overridden by condition namespace",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root xmlns:ns="http://example.com">
    <ns:user>Grace</ns:user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//ns:user",
				XMLNamespaces: map[string]string{
					"ns": "http://example.com",
				},
			},
			systemNamespaces: map[string]string{
				"ns": "http://example.com/wrong",
			},
			want: true,
		},
		{
			name: "no match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user>Grace</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//admin",
			},
			want: false,
		},
		{
			name: "invalid XML",
			body: []byte(`invalid xml`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user",
			},
			want: false,
		},
		{
			name: "empty body",
			body: []byte(``),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user",
			},
			want: false,
		},
		{
			name: "regex match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user>Grace123</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value:    "Grace\\d+",
					Operator: "Matches",
				},
				XPath: "//user",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchXPath(tt.body, tt.condition, tt.systemNamespaces)
			assert.Equal(t, tt.want, got)
		})
	}
}
