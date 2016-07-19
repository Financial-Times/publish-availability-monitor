package content

import "testing"

func TestIsMarkedDeleted_True(t *testing.T) {
	if !wordpressContentMarkedDeletedTrue.IsMarkedDeleted() {
		t.Error("Expected True, the story IS marked deleted")
	}
}

func TestIsMarkedDeleted_False(t *testing.T) {
	if wordpressContentMarkedDeletedFalse.IsMarkedDeleted() {
		t.Error("Expected False, the wordPress article IS NOT marked deleted")
	}
}

var wordpressContentMarkedDeletedTrue = WordPressMessage{
	Status: "error", Error: "Not found.",
}

var wordpressContentMarkedDeletedFalse = WordPressMessage{
	Status: "ok", Post: Post{},
}
