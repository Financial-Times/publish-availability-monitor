package main

import (
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/stretchr/testify/assert"
)

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
			h := kafkaMessageHandler{}
			kafkaMessage := kafka.FTMessage{
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
	e2eTestUUIDs := []string{"e4d2885f-1140-400b-9407-921e1c7378cd"}
	log := logger.NewUPPLogger("publish-availability-monitor", "INFO")

	mh := NewKafkaMessageHandler(nil, nil, nil, nil, nil, e2eTestUUIDs, log)
	kmh := mh.(*kafkaMessageHandler)

	kafkaMessage := kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id": syntheticTID,
		},
	}

	assert.Equal(t, false, kmh.isIgnorableMessage(kafkaMessage))
}

func TestUnmarshalContent_InvalidMessageMissingHeader_Error(t *testing.T) {
	h := kafkaMessageHandler{}
	if _, err := h.unmarshalContent(invalidMessageWrongHeader); err == nil {
		t.Error("Expected failure, but message with missing system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_InvalidMessageWrongSystemId_Error(t *testing.T) {
	h := kafkaMessageHandler{}
	if _, err := h.unmarshalContent(invalidMessageWrongSystemID); err == nil {
		t.Error("Expected failure, but message with wrong system ID successfully unmarshalled!")
	}
}

func TestUnmarshalContent_ValidVideoMessage(t *testing.T) {
	testCases := []struct {
		videoMessage kafka.FTMessage
	}{
		{validVideoMsg},
		{validVideoMsg2},
	}

	h := kafkaMessageHandler{}
	log := logger.NewUPPLogger("test", "PANIC")

	for _, testCase := range testCases {
		resultContent, err := h.unmarshalContent(testCase.videoMessage)
		if err != nil {
			t.Errorf("Expected success, but error occurred [%v]", err)
			return
		}
		valRes := resultContent.Validate("", "", "", "", log)
		assert.False(t, valRes.IsMarkedDeleted, "Expected published content.")
	}
}

func TestUnmarshalContent_ValidDeletedVideoMessage(t *testing.T) {
	h := kafkaMessageHandler{}
	resultContent, err := h.unmarshalContent(validDeleteVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occurred [%v]", err)
		return
	}
	log := logger.NewUPPLogger("test", "PANIC")

	valRes := resultContent.Validate("", "", "", "", log)
	assert.True(t, valRes.IsMarkedDeleted, "Expected deleted content.")
}

func TestUnmarshalContent_InvalidVideoMessage(t *testing.T) {
	h := kafkaMessageHandler{}
	resultContent, err := h.unmarshalContent(invalidVideoMsg)
	if err != nil {
		t.Errorf("Expected success, but error occurred [%v]", err)
		return
	}
	log := logger.NewUPPLogger("test", "PANIC")

	valRes := resultContent.Validate("", "", "", "", log)
	assert.False(t, valRes.IsValid, "Expected invalid content.")
}

func TestUnmarshalContent_VideoBinaryContentSet(t *testing.T) {
	h := kafkaMessageHandler{}
	resultContent, err := h.unmarshalContent(validVideoMsg)
	assert.NoError(t, err)

	video, ok := resultContent.(content.Video)
	assert.True(t, ok)

	assert.Equal(t, []byte(validVideoMsg.Body), video.BinaryContent)
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
	assert.False(t, genericContent.Deleted)
}

func TestUnmarshalContent_DeletedGenericContent(t *testing.T) {
	h := kafkaMessageHandler{}

	resultContent, err := h.unmarshalContent(validDeletedGenericContentMessage)
	assert.NoError(t, err)

	genericContent, ok := resultContent.(content.GenericContent)
	assert.True(t, ok)

	assert.Equal(t, "077f5ac2-0491-420e-a5d0-982e0f86204b", genericContent.GetUUID())
	assert.Equal(t, validDeletedGenericContentMessage.Headers["Content-Type"], genericContent.GetType())
	assert.Equal(t, []byte(validDeletedGenericContentMessage.Body), genericContent.BinaryContent)
	assert.True(t, genericContent.Deleted)
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

var invalidMessageWrongHeader = kafka.FTMessage{
	Headers: map[string]string{
		"Foobar-System-Id": "http://cmdb.ft.com/systems/cct",
	},
	Body: "{}",
}
var invalidMessageWrongSystemID = kafka.FTMessage{
	Headers: map[string]string{
		"Origin-System-Id": "web-foobar",
	},
	Body: "{}",
}

var validVideoMsg = kafka.FTMessage{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"id": "e28b12f7-9796-3331-b030-05082f0b8157"
	}`,
}

var validVideoMsg2 = kafka.FTMessage{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"id": "e28b12f7-9796-3331-b030-05082f0b8157",
		"deleted": false
	}`,
}

var validDeleteVideoMsg = kafka.FTMessage{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"id": "e28b12f7-9796-3331-b030-05082f0b8157",
		"deleted": true
	}`,
}

var invalidVideoMsg = kafka.FTMessage{
	Headers: map[string]string{
		"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
	},
	Body: `{
		"uuid": "e28b12f7-9796-3331-b030-05082f0b8157",
		"something_else": "something else"
	}`,
}

var validGenericContentMessage = kafka.FTMessage{
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

var validDeletedGenericContentMessage = kafka.FTMessage{
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
		"deleted": true
	  }`,
}

var validGenericAudioMessage = kafka.FTMessage{
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
