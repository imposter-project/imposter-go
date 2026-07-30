package template

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTemplateExchange builds a minimal exchange with an empty GET request and
// the supplied request store, matching the pattern used elsewhere in this
// package's tests.
func newTemplateExchange(requestStore *store.Store) *exchange.Exchange {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	body := []byte("")
	req.Body = io.NopCloser(bytes.NewReader(body))
	return exchange.NewExchangeFromRequest(req, body, requestStore)
}

func TestDatetimeReplacement(t *testing.T) {
	exch := newTemplateExchange(store.NewRequestStore())
	cfg := &config.ImposterConfig{ServerPort: "8080"}

	t.Run("iso8601_date", func(t *testing.T) {
		got := ProcessTemplate("${datetime.now.iso8601_date}", exch, cfg, &config.RequestMatcher{})
		assert.Regexp(t, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`), got)
	})

	t.Run("iso8601_datetime is RFC3339", func(t *testing.T) {
		got := ProcessTemplate("${datetime.now.iso8601_datetime}", exch, cfg, &config.RequestMatcher{})
		_, err := time.Parse(time.RFC3339, got)
		require.NoError(t, err, "expected an RFC3339 timestamp, got %q", got)
	})

	t.Run("millis and nanos are numeric", func(t *testing.T) {
		millis := ProcessTemplate("${datetime.now.millis}", exch, cfg, &config.RequestMatcher{})
		nanos := ProcessTemplate("${datetime.now.nanos}", exch, cfg, &config.RequestMatcher{})
		assert.Regexp(t, regexp.MustCompile(`^\d+$`), millis)
		assert.Regexp(t, regexp.MustCompile(`^\d+$`), nanos)
	})

	t.Run("unknown field yields empty", func(t *testing.T) {
		got := ProcessTemplate("${datetime.now.unknown}", exch, cfg, &config.RequestMatcher{})
		assert.Equal(t, "", got)
	})

	t.Run("unknown subcategory yields empty", func(t *testing.T) {
		got := ProcessTemplate("${datetime.later.millis}", exch, cfg, &config.RequestMatcher{})
		assert.Equal(t, "", got)
	})
}

func TestStoreReplacement_ComplexValue(t *testing.T) {
	cfg := &config.ImposterConfig{ServerPort: "8080"}

	t.Run("non-string value is JSON encoded", func(t *testing.T) {
		s := store.NewRequestStore()
		s.StoreValue("payload", map[string]interface{}{"active": true})
		exch := newTemplateExchange(s)

		got := ProcessTemplate("${stores.request.payload}", exch, cfg, &config.RequestMatcher{})
		assert.JSONEq(t, `{"active":true}`, got)
	})

	t.Run("string value is returned verbatim", func(t *testing.T) {
		s := store.NewRequestStore()
		s.StoreValue("name", "Grace")
		exch := newTemplateExchange(s)

		got := ProcessTemplate("${stores.request.name}", exch, cfg, &config.RequestMatcher{})
		assert.Equal(t, "Grace", got)
	})
}

func TestSystemReplacement_EdgeCases(t *testing.T) {
	exch := newTemplateExchange(store.NewRequestStore())

	t.Run("nil imposter config yields empty", func(t *testing.T) {
		got := ProcessTemplate("${system.server.port}", exch, nil, &config.RequestMatcher{})
		assert.Equal(t, "", got)
	})

	t.Run("unknown subcategory yields empty", func(t *testing.T) {
		cfg := &config.ImposterConfig{ServerPort: "8080"}
		got := ProcessTemplate("${system.client.port}", exch, cfg, &config.RequestMatcher{})
		assert.Equal(t, "", got)
	})

	t.Run("unknown field yields empty", func(t *testing.T) {
		cfg := &config.ImposterConfig{ServerPort: "8080"}
		got := ProcessTemplate("${system.server.hostname}", exch, cfg, &config.RequestMatcher{})
		assert.Equal(t, "", got)
	})
}
