package content

import (
	"testing"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalContent_ValidMessageMethodeSystemHeader_NoError(t *testing.T) {
	if _, err := UnmarshalContent(validMethodeMessage); err != nil {
		t.Errorf("Message with valid system ID [%s] cannot be unmarshalled!", validMethodeMessage.Headers["Origin-System-Id"])
	}
}

func TestUnmarshalContent_ValidMessageWordpressSystemHeader_NoError(t *testing.T) {
	if _, err := UnmarshalContent(validWordpressMessage); err != nil {
		t.Errorf("Message with valid system ID [%s] cannot be unmarshalled!", validWordpressMessage.Headers["Origin-System-Id"])
	}
}

func TestUnmarshalContent_InvalidMessageMissingHeader_Error(t *testing.T) {
	if _, err := UnmarshalContent(invalidMessageWrongHeader); err == nil {
		t.Error("Expected failure, but message with missing system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidMessageWrongSystemId_Error(t *testing.T) {
	if _, err := UnmarshalContent(invalidMessageWrongSystemID); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidMethodeContentWrongJSONFormat_Error(t *testing.T) {
	if _, err := UnmarshalContent(invalidMethodeMessageWrongJSONFormat); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidWordPressContentWrongJSONFormat_Error(t *testing.T) {
	if _, err := UnmarshalContent(invalidWordPressMessageWrongJSONFormat); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_ValidWordPressMessageWithTypeField_TypeIsCorrectlyUnmarshalled(t *testing.T) {
	content, err := UnmarshalContent(validWordPressMessageWithTypeField)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	if content.GetType() != "post" {
		t.Errorf("Expected [post] content type, but found [%s].", content.GetType())
	}
}

func TestUnmarshalContent_ValidVideoMessage(t *testing.T) {
	content, err := UnmarshalContent(validVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	valRes := content.Validate("", "", "", "")
	if valRes.IsMarkedDeleted {
		t.Errorf("Expected published content.")
	}
}

func TestUnmarshalContent_ValidDeletedVideoMessage(t *testing.T) {
	content, err := UnmarshalContent(validDeleteVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	valRes := content.Validate("", "", "", "")
	if !valRes.IsMarkedDeleted {
		t.Errorf("Expected deleted content.")
	}
}

func TestUnmarshalContent_InvalidVideoMessage(t *testing.T) {
	content, err := UnmarshalContent(invalidVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	valRes := content.Validate("", "", "", "")
	if valRes.IsValid {
		t.Errorf("Expected invalid content.")
	}
}

func TestUnmarshalContent_ContentIsMethodeList_LinkedObjectsFieldIsMarshalled(t *testing.T) {
	var validMethodeListMessage = consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
		},
		Body: string(loadBytesForFile(t, "methode_list.json")),
	}
	content, err := UnmarshalContent(validMethodeListMessage)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	methodeContent, ok := content.(EomFile)
	if !ok {
		t.Error("Expected Methode list to be an EomFile")
	}
	if len(methodeContent.LinkedObjects) == 0 {
		t.Error("Expected list to have several linked objects, but parsed none")
	}
}

func TestUnmarshalContent_ContentIsMethodeArticle_LinkedObjectsFieldIsEmpty(t *testing.T) {
	var validMethodeListMessage = consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
		},
		Body: string(loadBytesForFile(t, "methode_article.json")),
	}
	content, err := UnmarshalContent(validMethodeListMessage)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	methodeContent, ok := content.(EomFile)
	if !ok {
		t.Error("Expected Methode article to be an EomFile")
	}
	if len(methodeContent.LinkedObjects) != 0 {
		t.Error("Expected article to have zero linked objects, but found several")
	}
}

func TestUnmarshalContent_MethodeBinaryContentSet(t *testing.T) {
	content, err := UnmarshalContent(validMethodeMessage)
	assert.NoError(t, err)

	eomFile, ok := content.(EomFile)
	assert.True(t, ok)

	assert.Equal(t, []byte(validMethodeMessage.Body), eomFile.BinaryContent)
}

func TestUnmarshalContent_VideoBinaryContentSet(t *testing.T) {
	content, err := UnmarshalContent(validVideoMsg)
	assert.NoError(t, err)

	video, ok := content.(Video)
	assert.True(t, ok)

	assert.Equal(t, []byte(validVideoMsg.Body), video.BinaryContent)
}

func TestIsUUIDValid_UUIDValid(t *testing.T) {
	if !isUUIDValid(validUUID) {
		t.Error("Valid UUID marked as invalid!")
	}
}

func TestIsUUIDValid_UUIDInvalid(t *testing.T) {
	if isUUIDValid(invalidUUID) {
		t.Error("Invalid UUID marked as valid!")
	}
}

func TestIsValidContentPlaceholder(t *testing.T) {
	content, err := UnmarshalContent(validContentPlaceholder)
	if err != nil {
		t.Error("Expected failure")
		return
	}
	assert.Equal(t, "EOM::CompoundStory_ContentPlaceholder", content.GetType())

}

const validUUID = "e28b12f7-9796-3331-b030-05082f0b8157"
const invalidUUID = "foobar"

var invalidMethodeMessageWrongJSONFormat = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
	},
	Body: `{"uuid": "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", "type": "Image", "value" : " }`,
}

var validMethodeMessage = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
	},
	Body: "{}",
}

var invalidWordPressMessageWrongJSONFormat = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/wordpress",
	},
	Body: `{"status": "ok", "post": {"id : "002251", "type": "post"}}`,
}

var validWordPressMessageWithTypeField = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/wordpress",
	},
	Body: `{"status": "ok", "post": {"id" : "002251", "type": "post"}}`,
}

var validWordpressMessage = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/wordpress",
	},
	Body: "{}",
}
var invalidMessageWrongHeader = consumer.Message{
	Headers: map[string]string{
		"Foobar-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
	},
	Body: "{}",
}
var invalidMessageWrongSystemID = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "methode-web-foobar",
	},
	Body: "{}",
}

var validVideoMsg = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/brightcove",
	},
	Body: `{
		"uuid": "e28b12f7-9796-3331-b030-05082f0b8157",
		"id": "4966650664001",
		"name": "the-dark-knight.mp4",
		"published_at": "2016-06-01T21:40:19.120Z",
		"updated_at": "2016-06-01T21:40:19.120Z",
		"something_else": "something else"
	}`,
}

var validDeleteVideoMsg = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/brightcove",
	},
	Body: `{
		"uuid": "e28b12f7-9796-3331-b030-05082f0b8157",
		"id": "4966650664001",
		"name": "the-dark-knight.mp4",
		"something_else": "something else"
	}`,
}

var invalidVideoMsg = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/brightcove",
	},
	Body: `{
		"uuid": "e28b12f7-9796-3331-b030-05082f0b8157",
		"something_else": "something else"
	}`,
}

var validContentPlaceholder = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
	},
	Body: `{
		"uuid": "e28b12f7-9796-3331-b030-05082f0b8157",
		"type": "EOM::CompoundStory",
		"attributes": "<ObjectMetadata><EditorialNotes><Sources><Source><SourceCode>ContentPlaceholder</SourceCode></Source></Sources></EditorialNotes></ObjectMetadata>"
	}`,
}
