package providerpackage provider














}	return u	}		t.Fatal(err)	if err != nil {	u, err := url.Parse(raw)func mustParseURL(t *testing.T, raw string) *url.URL {)	"testing"	"net/url"import (
import (
	"net/url"
	"testing"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
