package content

import "testing"

func TestIsMarkedDeleted_True(t *testing.T) {
	valRes := wordpressContentMarkedDeletedTrue.Validate("", "", "", "")
	if !valRes.IsMarkedDeleted {
		t.Error("Expected True, the story IS marked deleted")
	}
}

func TestIsMarkedDeleted_False(t *testing.T) {
	valRes := wordpressContentMarkedDeletedFalse.Validate("", "", "", "")
	if valRes.IsMarkedDeleted {
		t.Error("Expected False, the wordPress article IS NOT marked deleted")
	}
}

var wordpressContentMarkedDeletedTrue = WordPressMessage{
	Status: "error", Error: "Not found.",
}

var wordpressContentMarkedDeletedFalse = WordPressMessage{
	Status: "ok", Post: Post{},
}

var wordpressContentWithValidBlogDomain = WordPressMessage{
	Status: "ok", Post: Post{"post", "58bdb656-8f7a-4a8c-b2b9-f9722824b318", "http://ftalphaville.ft.com/2016/10/28/2178195/firstft-eu-canada-trade-deal-salvaged-crimes-of-the-future-and-the-price-of-fast-fashion/"},
}

var wordpressContentWithInvalidBlogDomain = WordPressMessage{
	Status: "ok", Post: Post{"post", "58bdb656-8f7a-4a8c-b2b9-f9722824b318", "http://ftalphaville-wp.ft.com/?pid=1234"},
}

func TestIsValidBlogDomain_True(t *testing.T) {
	valRes := wordpressContentWithValidBlogDomain.Validate("", "", "", "")
	if !valRes.IsValid {
		t.Error("Expected True")
	}
}

func TestIsValidBlogDomain_False(t *testing.T) {
	valRes := wordpressContentWithInvalidBlogDomain.Validate("", "", "", "")
	if valRes.IsValid {
		t.Error("Expected False")
	}
}
