package botfilter

import "strings"

var botSubstrings = []string{
	"Googlebot",
	"bingbot",
	"Amazonbot",
	"GPTBot",
	"Bytespider",
	"Applebot",
	"Slurp",
	"DuckDuckBot",
	"facebookexternalhit",
	"Twitterbot",
	"LinkedInBot",
	"Baiduspider",
	"YandexBot",
	"Sogou",
	"MJ12bot",
	"AhrefsBot",
	"SemrushBot",
	"DotBot",
	"PetalBot",
	"curl/",
	"wget/",
	"python-requests",
	"Go-http-client",
	"HeadlessChrome",
	"PhantomJS",
	"Scrapy",
	"Crawl",
}

func IsBot(userAgent string) bool {
	if userAgent == "" {
		return true
	}
	lower := strings.ToLower(userAgent)
	for _, sub := range botSubstrings {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
