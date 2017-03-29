package content

import (
	"net/url"
	"strings"
)

type blogAPIEndpointMetadata struct {
	host string
}

func isValidBrand(requestURI string) bool {
	parsedURL, err := url.Parse(requestURI)
	if err != nil || parsedURL.Host == "" {
		warnLogger.Printf("Invalid request URI  [%s].", requestURI)
		return false
	}
	requestHostAndPath := parsedURL.Host + parsedURL.Path
	for _, blogAPIEndpointMetadata := range blogAPIEndpointMetadatas {
		if strings.Contains(requestHostAndPath, blogAPIEndpointMetadata.host) {
			return true
		}
	}
	return false
}

var blogAPIEndpointMetadatas = []blogAPIEndpointMetadata{
	{"blogs.ft.com/the-world"},
	{"blogs.ft.com/brusselsblog"},
	{"blogs.ft.com/businessblog"},
	{"blogs.ft.com/tech-blog"},
	{"blogs.ft.com/westminster"},
	{"ftalphaville.ft.com"},
	{"blogs.ft.com/mba-blog"},
	{"blogs.ft.com/beyond-brics"},
	{"blogs.ft.com/gavyndavies"},
	{"blogs.ft.com/material-world"},
	{"blogs.ft.com/ftdata"},
	{"blogs.ft.com/nick-butler"},
	{"blogs.ft.com/photo-diary"},
	{"blogs.ft.com/off-message"},
	{"blogs.ft.com/david-allen-green"},
	{"blogs.ft.com/andrew-smithers"},
	{"blogs.ft.com/lex-live"},
	{"blogs.ft.com/andrew-mcafee"},
	{"blogs.ft.com/the-exchange"},
	{"blogs.ft.com/larry-summers"},
	{"www.ft.com/fastft"},
	{"blogs.ft.com/fastft"},
}
