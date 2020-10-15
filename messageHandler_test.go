package main

import (
	"io/ioutil"
	"testing"

	consumer "github.com/Financial-Times/message-queue-gonsumer"
	"github.com/Financial-Times/publish-availability-monitor/v2/content"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const wordpressType = "wordpress"

const syntheticTID = "SYNTHETIC-REQ-MONe4d2885f-1140-400b-9407-921e1c7378cd"
const carouselRepublishTID = "tid_ofcysuifp0_carousel_1488384556"
const carouselUnconventionalRepublishTID = "republish_-10bd337c-66d4-48d9-ab8a-e8441fa2ec98_carousel_1493606135"
const carouselGeneratedTID = "tid_ofcysuifp0_carousel_1488384556_gentx"
const naturalTID = "tid_xltcnbckvq"

func TestIsIgnorableMessage_naturalMessage(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if h.isIgnorableMessage(naturalTID) {
		t.Error("Normal message marked as ignorable")
	}
}

func TestIsIgnorableMessage_syntheticMessage(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if !h.isIgnorableMessage(syntheticTID) {
		t.Error("Synthetic message marked as normal")
	}
}

func TestIsIgnorableMessage_carouselRepublishMessage(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if !h.isIgnorableMessage(carouselRepublishTID) {
		t.Error("Carousel republish message marked as normal")
	}
	if !h.isIgnorableMessage(carouselUnconventionalRepublishTID) {
		t.Error("Carousel republish message marked as normal")
	}
}

func TestIsIgnorableMessage_carouselGeneratedMessage(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if !h.isIgnorableMessage(carouselGeneratedTID) {
		t.Error("Carousel generated message marked as normal")
	}
}

func TestUnmarshalContent_ValidMessageMethodeSystemHeader_NoError(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUuid", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)

	h := kafkaMessageHandler{typeRes}

	if _, err := h.unmarshalContent(validMethodeMessage); err != nil {
		t.Errorf("Message with valid system ID [%s] cannot be unmarshalled!", validWordpressMessage.Headers["Origin-System-Id"])
	}
}

func TestUnmarshalContent_ValidMessageWordpressSystemHeader_NoError(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if _, err := h.unmarshalContent(validWordpressMessage); err != nil {
		t.Errorf("Message with valid system ID [%s] cannot be unmarshalled!", validWordpressMessage.Headers["Origin-System-Id"])
	}
}

func TestUnmarshalContent_InvalidMessageMissingHeader_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if _, err := h.unmarshalContent(invalidMessageWrongHeader); err == nil {
		t.Error("Expected failure, but message with missing system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidMessageWrongSystemId_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if _, err := h.unmarshalContent(invalidMessageWrongSystemID); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidMethodeContentWrongJSONFormat_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUuid", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes}

	if _, err := h.unmarshalContent(invalidMethodeMessageWrongJSONFormat); err == nil {
		t.Error("Expected failure, but message with wrong JSON format successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidWordPressContentWrongJSONFormat_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	if _, err := h.unmarshalContent(invalidWordPressMessageWrongJSONFormat); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_ValidWordPressMessageWithTypeField_TypeIsCorrectlyUnmarshalled(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	resultContent, err := h.unmarshalContent(validWordPressMessageWithTypeField)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	if resultContent.GetType() != wordpressType {
		t.Errorf("Expected [%s] content type, but found [%s].", wordpressType, resultContent.GetType())
	}
}

func TestUnmarshalContent_ValidVideoMessage(t *testing.T) {
	testCases := []struct {
		videoMessage consumer.Message
	}{
		{validVideoMsg},
		{validVideoMsg2},
	}

	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	for _, testCase := range testCases {
		resultContent, err := h.unmarshalContent(testCase.videoMessage)
		if err != nil {
			t.Errorf("Expected success, but error occured [%v]", err)
			return
		}
		valRes := resultContent.Validate("", "", "", "")
		assert.False(t, valRes.IsMarkedDeleted, "Expected published content.")
	}
}

func TestUnmarshalContent_ValidDeletedVideoMessage(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	resultContent, err := h.unmarshalContent(validDeleteVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	valRes := resultContent.Validate("", "", "", "")
	assert.True(t, valRes.IsMarkedDeleted, "Expected deleted content.")
}

func TestUnmarshalContent_InvalidVideoMessage(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	resultContent, err := h.unmarshalContent(invalidVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	valRes := resultContent.Validate("", "", "", "")
	assert.False(t, valRes.IsValid, "Expected invalid content.")
}

func TestUnmarshalContent_ContentIsMethodeList_LinkedObjectsFieldIsMarshalled(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUuid", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes}

	var validMethodeListMessage = consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
			"X-Request-Id":     "tid_0123wxyz",
		},
		Body: string(loadBytesForFile(t, "content/testdata/methode_list.json")),
	}
	resultContent, err := h.unmarshalContent(validMethodeListMessage)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	methodeContent, ok := resultContent.(content.EomFile)
	if !ok {
		t.Error("Expected Methode list to be an EomFile")
	}
	if len(methodeContent.LinkedObjects) == 0 {
		t.Error("Expected list to have several linked objects, but parsed none")
	}
}

func TestUnmarshalContent_ContentIsMethodeArticle_LinkedObjectsFieldIsEmpty(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUuid", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes}

	var validMethodeListMessage = consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
			"X-Request-Id":     "tid_0123wxyz",
		},
		Body: string(loadBytesForFile(t, "content/testdata/methode_article.json")),
	}
	resultContent, err := h.unmarshalContent(validMethodeListMessage)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	methodeContent, ok := resultContent.(content.EomFile)
	if !ok {
		t.Error("Expected Methode article to be an EomFile")
	}
	if len(methodeContent.LinkedObjects) != 0 {
		t.Error("Expected article to have zero linked objects, but found several")
	}
}

func TestUnmarshalContent_ContentIsMethodeList_EmptyLinkedObjectsFieldIsMarshalled(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUuid", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes}

	var validMethodeListMessage = consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
			"X-Request-Id":     "tid_0123wxyz",
		},
		Body: string(loadBytesForFile(t, "content/testdata/methode_empty_list.json")),
	}
	resultContent, err := h.unmarshalContent(validMethodeListMessage)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	methodeContent, ok := resultContent.(content.EomFile)
	if !ok {
		t.Error("Expected Methode list to be an EomFile")
	}
	if len(methodeContent.LinkedObjects) != 0 {
		t.Error("Expected list to have no linked objects")
	}
}

func TestUnmarshalContent_MethodeBinaryContentSet(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUuid", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes}

	resultContent, err := h.unmarshalContent(validMethodeMessage)
	assert.NoError(t, err)

	eomFile, ok := resultContent.(content.EomFile)
	assert.True(t, ok)

	assert.Equal(t, []byte(validMethodeMessage.Body), eomFile.BinaryContent)
}

func TestUnmarshalContent_VideoBinaryContentSet(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes}

	resultContent, err := h.unmarshalContent(validVideoMsg)
	assert.NoError(t, err)

	video, ok := resultContent.(content.Video)
	assert.True(t, ok)

	assert.Equal(t, []byte(validVideoMsg.Body), video.BinaryContent)
}

func TestIsValidExternalCPH(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUuid", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory_External_CPH", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes}

	_, err := h.unmarshalContent(validContentPlaceholder)
	if err != nil {
		t.Error("Valid external CPH shouldn't throw error.")
	}
}

var invalidMethodeMessageWrongJSONFormat = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
		"X-Request-Id":     "tid_0123wxyz",
	},
	Body: `{"uuid": "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", "type": "Image", "value": " }`,
}

var validMethodeMessage = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
		"X-Request-Id":     "tid_0123wxyz",
	},
	Body: `{"uuid": "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", "type": "EOM::CompoundStory", "value": "" }`,
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
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"id": "e28b12f7-9796-3331-b030-05082f0b8157"
	}`,
}

var validVideoMsg2 = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"id": "e28b12f7-9796-3331-b030-05082f0b8157",
		"deleted": false
	}`,
}

var validDeleteVideoMsg = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"id": "e28b12f7-9796-3331-b030-05082f0b8157",
		"deleted": true
	}`,
}

var invalidVideoMsg = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"uuid": "e28b12f7-9796-3331-b030-05082f0b8157",
		"something_else": "something else"
	}`,
}

var validContentPlaceholder = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
		"X-Request-Id":     "tid_0123wxyz",
	},
	Body: `{
		"uuid": "e28b12f7-9796-3331-b030-05082f0b8157",
		"type": "EOM::CompoundStory",
		"attributes": "<ObjectMetadata><EditorialNotes><Sources><Source><SourceCode>ContentPlaceholder</SourceCode></Source></Sources></EditorialNotes></ObjectMetadata>"
	}`,
}

type MockTypeResolver struct {
	mock.Mock
}

func (m *MockTypeResolver) ResolveTypeAndUuid(eomFile content.EomFile, txID string) (string, string, error) {
	args := m.Called(eomFile, txID)
	return args.String(0), args.String(1), args.Error(2)
}

func loadBytesForFile(t *testing.T, path string) []byte {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
