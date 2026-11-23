package prometheus

import (
	"encoding/json"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	for _, raw := range []string{
		`{}`, `null`, `{"base_url":"file:///tmp/metrics"}`,
		`{"base_url":"http://localhost:9090","step":"0s"}`,
		`{"base_url":"http://localhost:9090","step":"1.5s"}`,
		`{"base_url":"http://localhost:9090","window":"-1h"}`,
		`{"base_url":"http://localhost:9090","window":"1m","step":"2m"}`,
		`{"base_url":"http://localhost:9090","typo":true}`,
		`{"base_url":"http://localhost:9090","transport":{"auth":{"bearer_token_env":"PROM_TOKEN"}}}`,
		`{"base_url":"http://localhost:9090","transport":{"auth":{"password_file":"/run/secrets/password"}}}`,
		`{"base_url":"http://localhost:9090","mapping":{"typo":"cluster"}}`,
	} {
		if _, err := FromJSON(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if _, err := FromJSON(json.RawMessage(`{"base_url":"http://localhost:9090"}`)); err != nil {
		t.Fatal(err)
	}
}
