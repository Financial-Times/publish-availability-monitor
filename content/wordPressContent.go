package content

var validWordPressTypes []string

func init() {
	validWordPressTypes = []string{
		"post",
		"webchat-markets-live",
		"webchat-live-blogs",
		"webchat-live-qa",
	}
}

const notFoundError = "Not found."

// WordPressMessage models messages from Wordpress
type WordPressMessage struct {
	Status      string `json:"status"`
	APIURL      string `json:"apiUrl"`
	Error       string `json:"error"`
	Post        Post   `json:"post"`
	PreviousURL string `json:"previousUrl"`
}

// Post models WordPress content
// neglect unused fields (e.g. id, slug, title, content, etc)
type Post struct {
	Type string `json:"type"`
	UUID string `json:"uuid"`
}

func (wordPressMessage WordPressMessage) IsValid(extValEndpoint string) bool {
	if wordPressMessage.Status == "error" && wordPressMessage.Error != notFoundError {
		//it's an error which we do not understand
		return false
	}

	contentUUID := wordPressMessage.Post.UUID
	if !isUUIDValid(contentUUID) {
		warnLogger.Printf("WordPress message invalid: invalid UUID: [%s]", contentUUID)
		return false
	}

	apiURL := wordPressMessage.APIURL
	if !isValidBrand(apiURL) {
		warnLogger.Printf("WordPress message invalid: failed to resolve brand for uri [%s].", apiURL)
		return false
	}

	contentType := wordPressMessage.Post.Type
	for _, validType := range validWordPressTypes {
		if contentType == validType {
			return true
		}
	}
	warnLogger.Printf("WordPress message invalid: unexpected content type: [%s]", contentType)
	return false
}

func (wordPressMessage WordPressMessage) IsMarkedDeleted() bool {
	if wordPressMessage.Status == "error" && wordPressMessage.Error == notFoundError {
		return true
	}
	return false
}

func (wordPressMessage WordPressMessage) GetType() string {
	return wordPressMessage.Post.Type
}

func (wordPressMessage WordPressMessage) GetUUID() string {
	return wordPressMessage.Post.UUID
}
