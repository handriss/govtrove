package botfilter

import "testing"

func TestIsBot(t *testing.T) {
	tests := []struct {
		ua   string
		want bool
	}{
		{"", true},
		{"Googlebot/2.1 (+http://www.google.com/bot.html)", true},
		{"Mozilla/5.0 (compatible; bingbot/2.0)", true},
		{"Amazonbot/0.1", true},
		{"Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.0)", true},
		{"facebookexternalhit/1.1", true},
		{"Twitterbot/1.0", true},
		{"curl/7.68.0", true},
		{"wget/1.21", true},
		{"python-requests/2.28.0", true},
		{"Go-http-client/1.1", true},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36", false},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36", false},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)", false},
	}

	for _, tt := range tests {
		t.Run(tt.ua, func(t *testing.T) {
			if got := IsBot(tt.ua); got != tt.want {
				t.Errorf("IsBot(%q) = %v, want %v", tt.ua, got, tt.want)
			}
		})
	}
}
