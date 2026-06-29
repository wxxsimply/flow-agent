package llm

import "testing"

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		err  string
		want bool
	}{
		{"deepseek: decode response: unexpected end of JSON input", true},
		{"deepseek: empty response body (status 200 OK)", true},
		{"deepseek: empty message content", true},
		{"bailian: 429 Too Many Requests", true},
		{"deepseek: 401 Unauthorized", false},
		{"invalid api key", false},
	}
	for _, c := range cases {
		if got := IsRetryable(fmtError(c.err)); got != c.want {
			t.Fatalf("%q: got %v want %v", c.err, got, c.want)
		}
	}
}

type fmtError string

func (e fmtError) Error() string { return string(e) }
