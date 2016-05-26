package main

import "testing"

func TestBuildHealthURL(t *testing.T) {
	var testCases = []struct {
		validationURL     string
		expectedHealthURL string
	}{
		{
			validationURL:     "http://methode-article-transformer/content-transformer/",
			expectedHealthURL: "http://methode-article-transformer/__health",
		},
		{
			validationURL:     "http://methode-article-transformer/content-transformer",
			expectedHealthURL: "http://methode-article-transformer/__health",
		},
		{
			validationURL:     "http://methode-article-transformer/__methode-article-transformer/content-transformer",
			expectedHealthURL: "http://methode-article-transformer/__methode-article-transformer/__health",
		},
		{
			validationURL:     "http://methode-article-transformer",
			expectedHealthURL: "http://methode-article-transformer/__health",
		},
		{
			validationURL:     "http://methode-article-transformer:8080",
			expectedHealthURL: "http://methode-article-transformer:8080/__health",
		},
		{
			validationURL:     "http://localhost:8080/__methode-article-transformer/content-transformer/",
			expectedHealthURL: "http://localhost:8080/__methode-article-transformer/__health",
		},
	}
	for _, tc := range testCases {
		if actual, _ := buildHealthURL(tc.validationURL); actual != tc.expectedHealthURL {
			t.Errorf("Expected: [%s]\nActual: [%s]", tc.expectedHealthURL, actual)
		}

	}
}
