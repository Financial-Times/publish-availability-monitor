package main

import (
	"io/ioutil"
	"testing"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const wordpressType = "wordpress"
const syntheticTID = "SYNTHETIC-REQ-MONe4d2885f-1140-400b-9407-921e1c7378cd"
const carouselRepublishTID = "tid_ofcysuifp0_carousel_1488384556"
const carouselUnconventionalRepublishTID = "republish_-10bd337c-66d4-48d9-ab8a-e8441fa2ec98_carousel_1493606135"
const carouselGeneratedTID = "tid_ofcysuifp0_carousel_1488384556_gentx"
const naturalTID = "tid_xltcnbckvq"

func TestIsIgnorableMessage(t *testing.T) {
	tests := map[string]struct {
		TransactionID  string
		ExpectedResult bool
	}{
		"normal message should not be ignored": {
			TransactionID:  naturalTID,
			ExpectedResult: false,
		},
		"synthetic message should be ignored": {
			TransactionID:  syntheticTID,
			ExpectedResult: true,
		},
		"carousel republish message should be ignored": {
			TransactionID:  carouselRepublishTID,
			ExpectedResult: true,
		},
		"carousel unconventional republish message should be ignored": {
			TransactionID:  carouselUnconventionalRepublishTID,
			ExpectedResult: true,
		},
		"carousel generated message should be ignored": {
			TransactionID:  carouselGeneratedTID,
			ExpectedResult: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			typeRes := new(MockTypeResolver)
			h := kafkaMessageHandler{typeRes: typeRes}
			kafkaMessage := consumer.Message{
				Headers: map[string]string{
					"X-Request-Id": test.TransactionID,
				},
			}

			got := h.isIgnorableMessage(kafkaMessage)

			if got != test.ExpectedResult {
				t.Fatalf("expected %v, got %v", test.ExpectedResult, got)
			}
		})
	}
}

func TestTestIsIgnorableMessage_SyntheticE2ETest(t *testing.T) {
	typeRes := new(MockTypeResolver)
	e2eTestUUIDs := []string{"e4d2885f-1140-400b-9407-921e1c7378cd"}

	mh := NewKafkaMessageHandler(typeRes, nil, nil, nil, nil, nil, e2eTestUUIDs)
	kmh := mh.(*kafkaMessageHandler)

	kafkaMessage := consumer.Message{
		Headers: map[string]string{
			"X-Request-Id": syntheticTID,
		},
	}

	got := kmh.isIgnorableMessage(kafkaMessage)
	expected := false

	if got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestUnmarshalContent_ValidMessageMethodeSystemHeader_NoError(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUUID", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)

	h := kafkaMessageHandler{typeRes: typeRes}

	if _, err := h.unmarshalContent(validMethodeMessage); err != nil {
		t.Errorf("Message with valid system ID [%s] cannot be unmarshalled!", validWordpressMessage.Headers["Origin-System-Id"])
	}
}

func TestUnmarshalContent_ValidMessageWordpressSystemHeader_NoError(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes: typeRes}

	if _, err := h.unmarshalContent(validWordpressMessage); err != nil {
		t.Errorf("Message with valid system ID [%s] cannot be unmarshalled!", validWordpressMessage.Headers["Origin-System-Id"])
	}
}

func TestUnmarshalContent_InvalidMessageMissingHeader_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes: typeRes}

	if _, err := h.unmarshalContent(invalidMessageWrongHeader); err == nil {
		t.Error("Expected failure, but message with missing system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidMessageWrongSystemId_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes: typeRes}

	if _, err := h.unmarshalContent(invalidMessageWrongSystemID); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidMethodeContentWrongJSONFormat_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUUID", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes: typeRes}

	if _, err := h.unmarshalContent(invalidMethodeMessageWrongJSONFormat); err == nil {
		t.Error("Expected failure, but message with wrong JSON format successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidWordPressContentWrongJSONFormat_Error(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes: typeRes}

	if _, err := h.unmarshalContent(invalidWordPressMessageWrongJSONFormat); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_ValidWordPressMessageWithTypeField_TypeIsCorrectlyUnmarshalled(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes: typeRes}

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
	h := kafkaMessageHandler{typeRes: typeRes}

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
	h := kafkaMessageHandler{typeRes: typeRes}

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
	h := kafkaMessageHandler{typeRes: typeRes}

	resultContent, err := h.unmarshalContent(invalidVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occured [%v]", err)
		return
	}
	valRes := resultContent.Validate("", "", "", "")
	assert.False(t, valRes.IsValid, "Expected invalid content.")
}

func TestUnmarshalContent_ContentIsMethodeArticle_LinkedObjectsFieldIsEmpty(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUUID", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes: typeRes}

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

func TestUnmarshalContent_MethodeBinaryContentSet(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUUID", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes: typeRes}

	resultContent, err := h.unmarshalContent(validMethodeMessage)
	assert.NoError(t, err)

	eomFile, ok := resultContent.(content.EomFile)
	assert.True(t, ok)

	assert.Equal(t, []byte(validMethodeMessage.Body), eomFile.BinaryContent)
}

func TestUnmarshalContent_VideoBinaryContentSet(t *testing.T) {
	typeRes := new(MockTypeResolver)
	h := kafkaMessageHandler{typeRes: typeRes}

	resultContent, err := h.unmarshalContent(validVideoMsg)
	assert.NoError(t, err)

	video, ok := resultContent.(content.Video)
	assert.True(t, ok)

	assert.Equal(t, []byte(validVideoMsg.Body), video.BinaryContent)
}

func TestIsValidExternalCPH(t *testing.T) {
	typeRes := new(MockTypeResolver)
	typeRes.On("ResolveTypeAndUUID", mock.MatchedBy(func(eomFile content.EomFile) bool { return true }), "tid_0123wxyz").Return("EOM::CompoundStory_External_CPH", "79e7f5ed-63c7-46b2-9767-736f8ae3a3f6", nil)
	h := kafkaMessageHandler{typeRes: typeRes}

	_, err := h.unmarshalContent(validContentPlaceholder)
	if err != nil {
		t.Error("Valid external CPH shouldn't throw error.")
	}
}

func TestUnmarshalContent_GenericContent(t *testing.T) {
	h := kafkaMessageHandler{}

	resultContent, err := h.unmarshalContent(validGenericContentMessage)
	assert.NoError(t, err)

	genericContent, ok := resultContent.(content.GenericContent)
	assert.True(t, ok)

	assert.Equal(t, "077f5ac2-0491-420e-a5d0-982e0f86204b", genericContent.UUID)
	assert.Equal(t, validGenericContentMessage.Headers["Content-Type"], genericContent.Type)
	assert.Equal(t, []byte(validGenericContentMessage.Body), genericContent.BinaryContent)
}

func TestUnmarshalContent_GenericContent_Audio(t *testing.T) {
	h := kafkaMessageHandler{}

	resultContent, err := h.unmarshalContent(validGenericAudioMessage)
	assert.NoError(t, err)

	genericContent, ok := resultContent.(content.GenericContent)
	assert.True(t, ok)

	assert.Equal(t, "be003650-6c8f-4665-8640-9cbf292bb580", genericContent.UUID)
	assert.Equal(t, validGenericAudioMessage.Headers["Content-Type"], genericContent.Type)
	assert.Equal(t, []byte(validGenericAudioMessage.Body), genericContent.BinaryContent)
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

var validGenericContentMessage = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/cct",
		"X-Request-Id":     "tid_0123wxyz",
		"Content-Type":     "application/vnd.ft-upp-article-internal",
	},
	Body: `{
		"uuid": "077f5ac2-0491-420e-a5d0-982e0f86204b",
		"title": "A title",
		"type": "Article",
		"byline": "A byline",
		"identifiers": [
		  {
			"authority": "an authority",
			"identifierValue": "some identifier value"
		  },
		  {
			"authority": "another authority",
			"identifierValue": "some other identifier value"
		  }
		],
		"publishedDate": "2014-12-23T20:45:54.000Z",
		"firstPublishedDate": "2014-12-22T20:45:54.000Z",
		"bodyXML": "<body>Lorem ipsum</body>",
		"editorialDesk": "some string editorial desk identifier",
		"description": "Some descriptive explanation for this content",
		"mainImage": "0000aa3c-0056-506b-2b73-ed90e21b3e64",
		"standout": {
		  "editorsChoice": false,
		  "exclusive": false,
		  "scoop": false
		},
		"webUrl": "http://some.external.url.com/content",
		"canBeSyndicated" : "verify",
		"accessLevel" : "premium",
		"canBeDistributed": "no",
		"someUnknownProperty" : " is totally fine, we don't validate for unknown fields/properties"
	  }`,
}

var validGenericAudioMessage = consumer.Message{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
		"X-Request-Id":     "tid_0123wxyz",
		"Content-Type":     "application/vnd.ft-upp-audio",
	},
	Body: `{
		"title":"Is bitcoin a fraud?",
		"byline":"News features and analysis from Financial Times reporters around the world. FT News in Focus is produced by Fiona Symon.",
		"canBeSyndicated":"yes",
		"firstPublishedDate":"2017-09-19T17:58:51.000Z",
		"publishedDate":"2017-09-19T17:58:51.000Z",
		"uuid":"be003650-6c8f-4665-8640-9cbf292bb580",
		"type":"Audio",
		"body":"<body><p>The value of bitcoin fell sharply last week after Jamie Dimon, head of JPMorgan Chase, suggested the digital currency craze would suffer the same fate as the tulip mania of the 17th century. Patrick Jenkins discusses whether he is right with the FT's Laura Noonan and Izabella Kaminska. Music by Kevin MacLeod</p></body>",
		"mainImage":"56e9f220-6c64-4b6a-afcd-b57e1ceec807",
		"identifiers":[  
			{  
				"authority":"http://www.acast.com/",
				"identifierValue":"https://rss.acast.com/ft-news"
			},
			{  
				"authority":"http://www.acast.com/",
				"identifierValue":"https://www.acast.com/ft-news/isbitcoinafraud-"
			},
			{  
				"authority":"http://api.ft.com/system/NEXT-VIDEO-EDITOR",
				"identifierValue":"be003650-6c8f-4665-8640-9cbf292bb580"
			}
		],
		"dataSource":[  
			{  
				"binaryUrl":"https://media.acast.com/ft-news/isbitcoinafraud-/media.mp3",
				"duration":412000,
				"mediaType":"audio/mpeg"
			}
		]
	  }`,
}

type MockTypeResolver struct {
	mock.Mock
}

func (m *MockTypeResolver) ResolveTypeAndUUID(eomFile content.EomFile, txID string) (string, string, error) {
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
