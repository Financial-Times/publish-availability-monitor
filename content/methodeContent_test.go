package content

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const validUUID = "e28b12f7-9796-3331-b030-05082f0b8157"

func TestIsEomfileValid_EmptyValidationURL_Invalid(t *testing.T) {
	valRes := eomfileWithInvalidContentType.Validate("", validUUID, "", "")
	if valRes.IsValid {
		t.Error("Eomfile with empty validation URL marked as valid")
	}
}

func TestIsEomfileValid_ExternalValidationTrue_Valid(t *testing.T) {
	txId := "1234"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "tid_pam_"+txId, r.Header.Get("X-Request-Id"), "transaction id")
		assert.Equal(t, "/content-transform", r.RequestURI, "Invalid external validation URL")
		//return OK
	}))
	defer ts.Close()
	valRes := validCompoundStory.Validate(ts.URL+"/content-transform", "tid_"+txId, "", "")
	if !valRes.IsValid {
		t.Error("Valid CompoundStory marked as invalid!")
	}
}

func TestIsEomfileValid_ExternalValidationFalseStatusCode418_Invalid(t *testing.T) {
	txId := "invalidstory"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "tid_pam_"+txId, r.Header.Get("X-Request-Id"), "transaction id")
		assert.Equal(t, "/content-transform", r.RequestURI, "Invalid external validation URL")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer ts.Close()
	valRes := validCompoundStory.Validate(ts.URL+"/content-transform", "tid_"+txId, "", "")
	if valRes.IsValid {
		t.Error("Valid CompoundStory regarded as invalid by external validation marked as valid!")
	}
}

func TestIsEomfileValid_ExternalValidationFalseStatusCode422_Invalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/map", r.RequestURI, "Invalid external validation URL")
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer ts.Close()
	valRes := validCompoundStory.Validate(ts.URL+"/map", "", "", "")
	if valRes.IsValid {
		t.Error("Valid CompoundStory regarded as invalid by external validation marked as valid!")
	}
}

var eomfileWithInvalidContentType = EomFile{
	UUID:             validUUID,
	Type:             "FOOBAR",
	Value:            "/9j/4QAYRXhpZgAASUkqAAgAAAAAAAAAAAAAAP/sABFEdWNr",
	Attributes:       "attributes",
	SystemAttributes: "systemAttributes",
}

var validCompoundStory = EomFile{
	UUID:             validUUID,
	Type:             "EOM::CompoundStory",
	ContentType:      "EOM::CompoundStory",
	Value:            contentWithHeadline,
	Attributes:       validFileTypeAttributes,
	SystemAttributes: systemAttributesWebChannel,
}

func TestIsMarkedDeleted_CompoundStory_True(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/map", r.RequestURI, "Invalid external validation URL")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	valRes := compoundStoryMarkedDeletedTrue.Validate(ts.URL+"/map", "", "", "")

	if !valRes.IsMarkedDeleted {
		t.Error("Expected True, the compound story IS marked deleted")
	}
}

func TestIsMarkedDeleted_CompoundStory_False(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/map", r.RequestURI, "Invalid external validation URL")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	valRes := compoundStoryMarkedDeletedFalse.Validate(ts.URL+"/map", "", "", "")

	if valRes.IsMarkedDeleted {
		t.Error("Expected False, the compound story IS NOT marked deleted")
	}
}

func TestIsMarkedDeleted_Story_True(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/map", r.RequestURI, "Invalid external validation URL")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	valRes := storyMarkedDeletedTrue.Validate(ts.URL+"/map", "", "", "")

	if !valRes.IsMarkedDeleted {
		t.Error("Expected True, the story IS marked deleted")
	}
}

func TestIsMarkedDeleted_Story_False(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/map", r.RequestURI, "Invalid external validation URL")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	valRes := storyMarkedDeletedFalse.Validate(ts.URL+"/map", "", "", "")

	if valRes.IsMarkedDeleted {
		t.Error("Expected False, the story IS NOT marked deleted")
	}
}

func TestIsNotGoingToMarshallInternalApplicationFieldOfEomFile(t *testing.T) {
	expectedJSON := loadBytesForFile(t, "methode_article.json")
	var methodeArticle EomFile
	err := json.Unmarshal(expectedJSON, &methodeArticle)
	if err != nil {
		t.Error(err)
	}
	actualJSON, err := json.Marshal(methodeArticle)
	assert.NoError(t, err, "No errors in marshalling")
	assert.JSONEq(t, string(expectedJSON), string(actualJSON), "The internal fields should not appear")
}

var compoundStoryMarkedDeletedTrue = EomFile{
	UUID:             UUIDString,
	Type:             "EOM::CompoundStory",
	Value:            content,
	Attributes:       attributesMarkedDeletedTrue,
	SystemAttributes: systemAttributes,
}

var storyMarkedDeletedTrue = EomFile{
	UUID:             UUIDString,
	Type:             "EOM::Story",
	Value:            content,
	Attributes:       attributesMarkedDeletedTrue,
	SystemAttributes: systemAttributes,
}

var compoundStoryMarkedDeletedFalse = EomFile{
	UUID:             UUIDString,
	Type:             "EOM::CompoundStory",
	Value:            content,
	Attributes:       attributesMarkedDeletedFalse,
	SystemAttributes: systemAttributes,
}

var storyMarkedDeletedFalse = EomFile{
	UUID:             UUIDString,
	Type:             "EOM::Story",
	Value:            content,
	Attributes:       attributesMarkedDeletedFalse,
	SystemAttributes: systemAttributes,
}

const UUIDString = "e28b12f7-9796-3331-b030-05082f0b8157"
const systemAttributes = "<props><productInfo><name>FTcom</name><issueDate>20150915</issueDate></productInfo><workFolder>/FT/WorldNews</workFolder><subFolder>UKNews</subFolder></props>"
const content = "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4NCjwhRE9DVFlQRSBkb2MgU1lTVEVNICIvU3lzQ29uZmlnL1J1bGVzL2Z0cHNpLmR0ZCI+DQo8P0VNLWR0ZEV4dCAvU3lzQ29uZmlnL1J1bGVzL2Z0cHNpL2Z0cHNpLmR0eD8+DQo8P0VNLXRlbXBsYXRlTmFtZSAvU3lzQ29uZmlnL1RlbXBsYXRlcy9GVC9CYXNlLVN0b3J5LnhtbD8+DQo8P3htbC1mb3JtVGVtcGxhdGUgL1N5c0NvbmZpZy9UZW1wbGF0ZXMvRlQvQmFzZS1TdG9yeS54cHQ/Pg0KPD94bWwtc3R5bGVzaGVldCB0eXBlPSJ0ZXh0L2NzcyIgaHJlZj0iL1N5c0NvbmZpZy9SdWxlcy9mdHBzaS9GVC9tYWlucmVwLmNzcyI/Pg0KPGRvYyB4bWw6bGFuZz0iZW4tdWsiPjxsZWFkIGlkPSJVMzIwMDExNzk2MDc0NzhGQkUiPjxsZWFkLWhlYWRsaW5lIGlkPSJVMzIwMDExNzk2MDc0NzhhMkQiPjxuaWQtdGl0bGU+PGxuPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgbmV3cyBpbiBkZXB0aCB0aXRsZSBoZXJlXT8+DQo8L2xuPg0KPC9uaWQtdGl0bGU+DQo8aW4tZGVwdGgtbmF2LXRpdGxlPjxsbj48P0VNLWR1bW15VGV4dCBbSW5zZXJ0IGluIGRlcHRoIG5hdiB0aXRsZSBoZXJlXT8+DQo8L2xuPg0KPC9pbi1kZXB0aC1uYXYtdGl0bGU+DQo8aGVhZGxpbmUgaWQ9IlUzMjAwMTE3OTYwNzQ3OHNtQyI+PGxuPlRoaXMgaXMgYSBoZWFkbGluZQ0KPC9sbj4NCjwvaGVhZGxpbmU+DQo8c2t5Ym94LWhlYWRsaW5lIGlkPSJVMzIwMDExNzk2MDc0Nzg1a0YiPjxsbj48P0VNLWR1bW15VGV4dCBbU2t5Ym94IGhlYWRsaW5lIGhlcmVdPz4NCjwvbG4+DQo8L3NreWJveC1oZWFkbGluZT4NCjx0cmlwbGV0LWhlYWRsaW5lPjxsbj48P0VNLWR1bW15VGV4dCBbVHJpcGxldCBoZWFkbGluZSBoZXJlXT8+DQo8L2xuPg0KPC90cmlwbGV0LWhlYWRsaW5lPg0KPHByb21vYm94LXRpdGxlPjxsbj48P0VNLWR1bW15VGV4dCBbUHJvbW9ib3ggdGl0bGUgaGVyZV0/Pg0KPC9sbj4NCjwvcHJvbW9ib3gtdGl0bGU+DQo8cHJvbW9ib3gtaGVhZGxpbmU+PGxuPjw/RU0tZHVtbXlUZXh0IFtQcm9tb2JveCBoZWFkbGluZSBoZXJlXT8+DQo8L2xuPg0KPC9wcm9tb2JveC1oZWFkbGluZT4NCjxlZGl0b3ItY2hvaWNlLWhlYWRsaW5lIGlkPSJVMzIwMDExNzk2MDc0NzhGVkUiPjxsbj48P0VNLWR1bW15VGV4dCBbU3RvcnkgcGFja2FnZSBoZWFkbGluZV0/Pg0KPC9sbj4NCjwvZWRpdG9yLWNob2ljZS1oZWFkbGluZT4NCjxuYXYtY29sbGVjdGlvbi1oZWFkbGluZT48bG4+PD9FTS1kdW1teVRleHQgW05hdiBjb2xsZWN0aW9uIGhlYWRsaW5lXT8+DQo8L2xuPg0KPC9uYXYtY29sbGVjdGlvbi1oZWFkbGluZT4NCjxpbi1kZXB0aC1uYXYtaGVhZGxpbmU+PGxuPjw/RU0tZHVtbXlUZXh0IFtJbiBkZXB0aCBuYXYgaGVhZGxpbmVdPz4NCjwvbG4+DQo8L2luLWRlcHRoLW5hdi1oZWFkbGluZT4NCjwvbGVhZC1oZWFkbGluZT4NCjx3ZWItaW5kZXgtaGVhZGxpbmUgaWQ9IlUzMjAwMTE3OTYwNzQ3OHRQRyI+PGxuPjw/RU0tZHVtbXlUZXh0IFtTaG9ydCBoZWFkbGluZV0/Pg0KPC9sbj4NCjwvd2ViLWluZGV4LWhlYWRsaW5lPg0KPHdlYi1zdGFuZC1maXJzdCBpZD0iVTMyMDAxMTc5NjA3NDc4bmVFIj48cD48P0VNLWR1bW15VGV4dCBbTG9uZyBzdGFuZGZpcnN0XT8+DQo8L3A+DQo8L3dlYi1zdGFuZC1maXJzdD4NCjx3ZWItc3ViaGVhZCBpZD0iVTMyMDAxMTc5NjA3NDc4dGpHIj48cD48P0VNLWR1bW15VGV4dCBbU2hvcnQgc3RhbmRmaXJzdF0/Pg0KPC9wPg0KPC93ZWItc3ViaGVhZD4NCjxlZGl0b3ItY2hvaWNlIGlkPSJVMzIwMDExNzk2MDc0NzhRWUUiPjw/RU0tZHVtbXlUZXh0IFtTdG9yeSBwYWNrYWdlIGh5cGVybGlua10/Pg0KPC9lZGl0b3ItY2hvaWNlPg0KPGxlYWQtdGV4dCBpZD0iVTMyMDAxMTc5NjA3NDc4T0VEIj48bGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgbGVhZCBib2R5IHRleHQgaGVyZSAtIG1pbiAxMzAgY2hhcnMsIG1heCAxNTAgY2hhcnNdPz4NCjwvcD4NCjwvbGVhZC1ib2R5Pg0KPHRyaXBsZXQtbGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgdHJpcGxldCBsZWFkIGJvZHkgYm9keSBoZXJlXT8+DQo8L3A+DQo8L3RyaXBsZXQtbGVhZC1ib2R5Pg0KPGNvbHVtbmlzdC1sZWFkLWJvZHk+PHA+PD9FTS1kdW1teVRleHQgW0luc2VydCBjb2x1bW5pc3QgbGVhZCBib2R5IGJvZHkgaGVyZV0/Pg0KPC9wPg0KPC9jb2x1bW5pc3QtbGVhZC1ib2R5Pg0KPHNob3J0LWJvZHk+PHA+PD9FTS1kdW1teVRleHQgW0luc2VydCBzaG9ydCBib2R5IGJvZHkgaGVyZV0/Pg0KPC9wPg0KPC9zaG9ydC1ib2R5Pg0KPHNreWJveC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgc2t5Ym94IGJvZHkgaGVyZV0/Pg0KPC9wPg0KPC9za3lib3gtYm9keT4NCjxwcm9tb2JveC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgcHJvbW9ib3ggYm9keSBoZXJlXT8+DQo8L3A+DQo8L3Byb21vYm94LWJvZHk+DQo8dHJpcGxldC1zaG9ydC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgdHJpcGxldCBzaG9ydCBib2R5IGhlcmVdPz4NCjwvcD4NCjwvdHJpcGxldC1zaG9ydC1ib2R5Pg0KPGVkaXRvci1jaG9pY2Utc2hvcnQtbGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgZWRpdG9ycyBjaG9pY2Ugc2hvcnQgbGVhZCBib2R5IGhlcmVdPz4NCjwvcD4NCjwvZWRpdG9yLWNob2ljZS1zaG9ydC1sZWFkLWJvZHk+DQo8bmF2LWNvbGxlY3Rpb24tc2hvcnQtbGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgbmF2IGNvbGxlY3Rpb24gc2hvcnQgbGVhZCBib2R5IGhlcmVdPz4NCjwvcD4NCjwvbmF2LWNvbGxlY3Rpb24tc2hvcnQtbGVhZC1ib2R5Pg0KPC9sZWFkLXRleHQ+DQo8bGVhZC1pbWFnZXMgaWQ9IlUzMjAwMTE3OTYwNzQ3OEFkQiI+PHdlYi1tYXN0ZXIgaWQ9IlUzMjAwMTE3OTYwNzQ3OFpSRiIvPg0KPHdlYi1za3lib3gtcGljdHVyZS8+DQo8d2ViLWFsdC1waWN0dXJlLz4NCjx3ZWItcG9wdXAtcHJldmlldyB3aWR0aD0iMTY3IiBoZWlnaHQ9Ijk2Ii8+DQo8d2ViLXBvcHVwLz4NCjwvbGVhZC1pbWFnZXM+DQo8aW50ZXJhY3RpdmUtY2hhcnQ+PD9FTS1kdW1teVRleHQgW0ludGVyYWN0aXZlLWNoYXJ0IGxpbmtdPz4NCjwvaW50ZXJhY3RpdmUtY2hhcnQ+DQo8L2xlYWQ+DQo8c3Rvcnk+PGhlYWRibG9jayBpZD0iVTMyMDAxMTc5NjA3NDc4d21IIj48aGVhZGxpbmUgaWQ9IlUzMjAwMTE3OTYwNzQ3OFgwSCI+PGxuPjw/RU0tZHVtbXlUZXh0IFtIZWFkbGluZV0/Pg0KPC9sbj4NCjwvaGVhZGxpbmU+DQo8L2hlYWRibG9jaz4NCjx0ZXh0IGlkPSJVMzIwMDExNzk2MDc0Nzh3WEQiPjxieWxpbmU+PGF1dGhvci1uYW1lPktpcmFuIFN0YWNleSwgRW5lcmd5IENvcnJlc3BvbmRlbnQ8L2F1dGhvci1uYW1lPg0KPC9ieWxpbmU+DQo8Ym9keT48cD5DT1NUIENPTVBBUklTSU9OIFdJVEggT1RIRVIgUkVHSU9OUzwvcD4NCjxwPjwvcD4NCjxwPlRvdmUgU3R1aHIgU2pvYmxvbSBzdG9vZCBpbiBmcm9udCBvZiAzMDAgb2lsIGluZHVzdHJ5IGluc2lkZXJzIGluIEFiZXJkZWVuIGxhc3Qgd2Vlaywgc2xpcHBlZCBpbnRvIHRoZSBsb2NhbCBTY290dGlzaCBkaWFsZWN0LCBhbmQgZ2F2ZSB0aGVtIG9uZSBjbGVhciBtZXNzYWdlOiDigJxLZWVwIHlvdXIgaGVpZC7igJ08L3A+DQo8cD5UaGUgTm9yd2VnaWFuIFN0YXRvaWwgZXhlY3V0aXZlIHdhcyBhZGRyZXNzaW5nIHRoZSBiaWVubmlhbCBPZmZzaG9yZSBFdXJvcGUgY29uZmVyZW5jZSwgZG9taW5hdGVkIHRoaXMgeWVhciBieSB0YWxrIG9mIHRoZSBmYWxsaW5nIG9pbCBwcmljZSBhbmQgdGhlIDxhIGhyZWY9Imh0dHA6Ly93d3cuZnQuY29tL2Ntcy9zLzAvNGYwNDQyZTAtYmNkMi0xMWU0LWE5MTctMDAxNDRmZWFiN2RlLmh0bWwjYXh6ejNsbllyTDN3YiIgdGl0bGU9Ik5vcnRoIFNlYSBvaWw6IFRoYXQgc2lua2luZyBmZWVsaW5nIC0gRlQiPmZ1dHVyZSBvZiBOb3J0aCBTZWEgcHJvZHVjdGlvbjwvYT4uIDwvcD4NCjxwPk92ZXIgdGhlIHBhc3QgMTQgbW9udGhzLCB0aGUgb2lsIHByaWNlIGhhcyBkcm9wcGVkIGZyb20gYWJvdXQgJDExNSBwZXIgYmFycmVsIHRvIGFyb3VuZCAkNTAuIE5vd2hlcmUgaGFzIHRoaXMgYmVlbiBmZWx0IG1vcmUga2Vlbmx5IHRoYW4gaW4gdGhlIE5vcnRoIFNlYSwgRk9SIERFQ0FERVMgYSBwb3dlcmhvdXNlIG9mIHRoZSBCcml0aXNoIGVjb25vbXksIGJ1dCBOT1cgYW4gYXJlYSB3aGVyZSBvaWwgZmllbGRzIGFyZSBnZXR0aW5nIG9sZGVyLCBuZXcgZGlzY292ZXJpZXMgcmFyZXIgYW5kIGNvc3RzIHN0ZWVwZXIuIE5vIHdvbmRlciBNcyBTdHVociBTam9ibG9tIGlzIHRyeWluZyB0byBzdG9wIGhlciBjb2xsZWFndWVzIGZyb20gcGFuaWNraW5nLjwvcD4NCjxwPkJ5IG5lYXJseSBhbnkgbWV0cmljLCBkb2luZyBidXNpbmVzcyBpbiB0aGUgTm9ydGggU2VhIGlzIG1vcmUgZGlmZmljdWx0IHRoYW4gYXQgYW55IG90aGVyIHBlcmlvZCBpbiBpdHMgNTAteWVhciBoaXN0b3J5LiBXaGlsZSB0aGUgb2lsIHByaWNlIGhhcyBiZWVuIGxvd2VyIOKAlCBpdCB0b3VjaGVkICQxMCBhIGJhcnJlbCBpbiAxOTg2IOKAlCBzdWNoIGEgcmFwaWQgc2x1bXAgaGFzIG5ldmVyIGJlZW4gc2VlbiB3aGVuIGNvc3RzIHdlcmUgc28gaGlnaCBhbmQgd2l0aCBvaWwgcHJvZHVjdGlvbiBhbHJlYWR5IGluIGRlY2xpbmUuPC9wPg0KPHA+SVMgUFJPRFVDVElPTiBJTiBERUNMSU5FIEJFQ0FVU0UgTk9SVEggU0VBIElTIFJVTk5JTkcgRFJZPyBCVVQgRE9FUyBUSElTIERPV05UVVJOIFJJU0sgTUFTU0lWRSBBQ0NFTEVSQVRJT04gT0YgVEhBVCBERUNMSU5FIEJFQ0FVU0UgRVhQTE9SQVRJT04vREVWRUxPUE1FTlQgSVMgQkVJTkcgQ1VSVEFJTEVEPC9wPg0KPHA+T25lIGV4ZWN1dGl2ZSBmcm9tIGFuIG9pbCBtYWpvciBzYXlzOiDigJxXZSBoYXZlIG5ldmVyIGJlZm9yZSBleHBlcmllbmNlZCBhbnl0aGluZyBsaWtlIHRoaXMgYmVmb3JlIOKAlCBldmVyeXRoaW5nIGlzIGhhcHBlbmluZyBhdCB0aGUgc2FtZSB0aW1lLuKAnTwvcD4NCjxwPkFjY29yZGluZyB0byBhbmFseXNpcyBieSBFd2FuIE11cnJheSBvZiB0aGUgY29ycG9yYXRlIGhlYWx0aCBtb25pdG9yIENvbXBhbnkgV2F0Y2gsIG91dCBvZiB0aGUgMTI2IG9pbCBleHBsb3JhdGlvbiBhbmQgcHJvZHVjdGlvbiBjb21wYW5pZXMgbGlzdGVkIGluIExvbmRvbiwgNzcgcGVyIGNlbnQgYXJlIG5vdyBsb3NzLW1ha2luZy4gQVJFIEFMTCBUSEVTRSBJTiBUSEUgTk9SVEggU0VBPz8/IFRvdGFsIGxvc3NlcyBzdGFuZCBhdCDCozYuNmJuLjwvcD4NCjxwPjxhIGhyZWY9Imh0dHA6Ly9vaWxhbmRnYXN1ay5jby51ay9lY29ub21pYy1yZXBvcnQtMjAxNS5jZm0iIHRpdGxlPSJPaWwgJmFtcDsgR2FzIEVjb25vbWljIFJlcG9ydCAyMDE1Ij5GaWd1cmVzIHJlbGVhc2VkIGxhc3Qgd2VlayBieSBPaWwgJmFtcDsgR2FzIFVLPC9hPiwgdGhlIGluZHVzdHJ5IGJvZHksIHNob3cgdGhhdCBmb3IgdGhlIGZpcnN0IHRpbWUgc2luY2UgMTk3Nywgd2hlbiBOb3J0aCBTZWEgb2lsIHdhcyBzdGlsbCB5b3VuZywgaXQgbm93IGNvc3RzIG1vcmUgdG8gcHVtcCBvaWwgb3V0IG9mIHRoZSBzZWFiZWQgdGhhbiBpdCBnZW5lcmF0ZXMgaW4gcG9zdC10YXggcmV2ZW51ZT8/PyBQUk9GSVQ/Pz8uIDwvcD4NCjxwPjxhIGhyZWY9Imh0dHA6Ly9wdWJsaWMud29vZG1hYy5jb20vcHVibGljL21lZGlhLWNlbnRyZS8xMjUyOTI1NCIgdGl0bGU9IkxvdyBvaWwgcHJpY2UgYWNjZWxlcmF0aW5nIGRlY29tbWlzc2lvbmluZyBpbiB0aGUgVUtDUyAtIFdvb2RNYWMiPmFuYWx5c2lzIHB1Ymxpc2hlZCBsYXN0IHdlZWsgYnkgV29vZCBNYWNrZW56aWU8L2E+c3VnZ2VzdHMgMTQwIGZpZWxkcyBtYXkgY2xvc2UgaW4gdGhlIG5leHQgZml2ZSB5ZWFycywgZXZlbiBpZiB0aGUgb2lsIHByaWNlIHJldHVybnMgdG8gJDg1IGEgYmFycmVsLjwvcD4NCjxwPkRpZmZlcmVudCBjb21wYW5pZXMgb3BlcmF0aW5nIGluIHRoZSBOb3J0aCBTZWEgaGF2ZSByZXNwb25kZWQgdG8gdGhpcyBpbiBkaWZmZXJlbnQgd2F5cy48L3A+DQo8cD5PaWwgbWFqb3JzIHRlbmQgdG8gaGF2ZSBhbiBleGl0IHJvdXRlLiBUaGV5IGhhdmUgY3V0IGpvYnMg4oCUIDUsNTAwIGhhdmUgYmVlbiBsb3N0IHNvIGZhciwgd2l0aCBleGVjdXRpdmVzIHdhcm5pbmcgb2YgYW5vdGhlciAxMCwwMDAgaW4gdGhlIGNvbWluZyB5ZWFycyDigJQgYW5kIGFyZSBub3cgdHJ5aW5nIHRvIFJFRFVDRSBPVEhFUiBDT1NUUyBTVUNIIEFTPz8/IGN1dCBjb3N0cy4gQnV0IGlmIGFsbCBlbHNlIGZhaWxzLCB0aGV5IGNhbiBzaW1wbHkgZXhpdCB0aGUgcmVnaW9uIGFuZCBmb2N1cyB0aGVpciBlZmZvcnRzIG9uIGNoZWFwZXIgZmllbGRzIGVsc2V3aGVyZSBpbiB0aGUgd29ybGQuPC9wPg0KPHA+RnJhbmNl4oCZcyBUb3RhbCBsYXN0IG1vbnRoIGFncmVlZCB0byBzZWxsIE5vcnRoIFNlYSA8YSBocmVmPSJodHRwOi8vd3d3LmZ0LmNvbS9jbXMvcy8wLzg5MWQ2ZTU4LTRjMTktMTFlNS1iNTU4LThhOTcyMjk3NzE4OS5odG1sI2F4enozbG5Zckwzd2IiIHRpdGxlPSJGcmVuY2ggb2lsIG1ham9yIFRvdGFsIHRvIHNlbGwgJDkwMG0gb2YgTm9ydGggU2VhIGFzc2V0cyAtIEZUIj5nYXMgdGVybWluYWxzIGFuZCBwaXBlbGluZXM8L2E+IGluIGEgJDkwMG0gZGVhbCwgd2hpbGUgRW9uLCB0aGUgR2VybWFuIHV0aWxpdHkgY29tcGFueSwgaXMgbG9va2luZyBmb3IgYnV5ZXJzIGZvciBzb21lIG9mIGl0cyBhc3NldHMgaW4gdGhlIHJlZ2lvbi48L3A+DQo8cD5TbWFsbGVyIGNvbXBhbmllcyBob3dldmVyIHRlbmQgbm90IHRvIGhhdmUgdGhhdCBvcHRpb24uIEZvciBzb21lLCB0aGUgb2lsIHByaWNlIHBsdW5nZSBtZWFucyBzcXVlZXppbmcgY29zdHMgYXMgbXVjaCBhcyBwb3NzaWJsZSBpbiBhbiBlZmZvcnQgdG8gc3RheSBhZmxvYXQgdW50aWwgaXQgcmVib3VuZHMuPC9wPg0KPHA+RE8gQ0FQRVggQ1VUIEZJUlNUIEJZIEUgTlFVRVNUPC9wPg0KPHA+VEhFTiBFWFBMQUlOIEZPQ1VTIE9OIEVYVFJBQ1RJTkcgTU9SRSBGUk9NIEZFVyBGSUVMRFMgSVQgSVMgRk9DVVNJTkcgT04sIEFORCBDVVRUSU5HIE9QRVg8L3A+DQo8cD48L3A+DQo8cD5FbnF1ZXN0LCBmb3IgZXhhbXBsZSwgaGFzIG1hZGUgYSB2aXJ0dWUgZnJvbSBnZXR0aW5nIG1vcmUgb2lsIG91dCBvZiBtYXR1cmUgZmllbGRzIHRoYW4gbWFqb3JzIGNhbi4gSXRzIFRoaXN0bGUgb2lsIGZpZWxkIHdhcyBvbmNlIG93bmVkIGJ5IEJQLCBidXQgYXMgcHJvZHVjdGlvbiBkZWNsaW5lZCwgRW5xdWVzdCBzdGVwcGVkIGluLCBhbmQgYnkgbGFzdCB5ZWFyIHRoZSBjb21wYW55IG1hbmFnZWQgdG8gZXh0cmFjdCAzbSBiYXJyZWxzIGZyb20gaXQgZm9yIHRoZSBmaXJzdCB0aW1lIHNpbmNlIDE5OTcuPC9wPg0KPHA+T25lIHdheSB0byBkbyBzbz8/Pz8gaXMgdG8gY3V0IGNvc3RzIGFnZ3Jlc3NpdmVseS4gRW5xdWVzdCBoYXMgbG9va2VkIGF0IGV2ZXJ5dGhpbmcgaW5jbHVkaW5nIHJlbG9jYXRpbmcgaXRzIGhlbGljb3B0ZXIgc2VydmljZSBpbiBhbiBlZmZvcnQgdG8gc2F2ZSBtb25leS4gQW5kIG1vcmUgaXMgbGlrZWx5IHRvIGNvbWUuPC9wPg0KPHA+QW1qYWQgQnNlc2l1LCB0aGUgY29tcGFueeKAmXMgY2hpZWYgZXhlY3V0aXZlLCBzYWlkOiDigJxUaGUgaW5kdXN0cnkgd2VudCB0aHJvdWdoIGxhc3QgeWVhcuKAmXMgY3ljbGUgaW4gYSBsaXR0bGUgYml0IG9mIGEgdGltaWQgbWFubmVyLiBUaGVyZSB3YXMgYSBmZWVsaW5nIHRoYXQgbWF5YmUgdGhpbmdzIGNhbiBjb21lIGJhY2sgbGlrZSB0aGV5IGRpZCBpbiAyMDA4IHRvIDIwMDk/Pz8sIGJ1dCB0aGF0IGhhc27igJl0IGhhcHBlbmVkLuKAnTwvcD4NCjxwPkJ1dCBmb3IgbWFueSBzbWFsbGVyIGNvbXBhbmllcywgY3V0dGluZyBjb3N0cyBoYXMgYWxzbyBpbnZvbHZlZCBjdXR0aW5nIGV4cGxvcmF0aW9uIGFuZCBkZXZlbG9wbWVudC4gT3V0IG9mIHRoZSAxMiBmaWVsZHMgaXQgb3ducyBpbiB3aGljaCBvaWwgaGFzIGJlZW4gZGlzY292ZXJlZCwgRW5xdWVzdCBpcyBjdXJyZW50bHkgZGV2ZWxvcGluZyBvbmx5IHR3by4gVGhpcyBpcyBhIHRyZW5kIG92ZXIgdGhlIGluZHVzdHJ5IGFzIGEgd2hvbGU6IGluIDIwMDggdGhlIGluZHVzdHJ5IGRyaWxsZWQgNDAgZXhwbG9yYXRpb24gd2VsbHMgaW4gdGhlIE5vcnRoIFNlYS4gTGFzdCB5ZWFyLCB0aGF0IGZpZ3VyZSB3YXMganVzdCAxNC48L3A+DQo8cD48L3A+DQo8cD5CVVQgU09NRSBDT01QQU5JRVMgQVJFIFBST0NFRURJTkcgV0lUSCBCSUcgUFJPSkVDVFM6IFNVQ0ggQVMgUFJFTUlFUiBBVCBTT0xBTjwvcD4NCjxwPklUIE1FQU5TIFNPTUUgQ09NUEFOSUVTIEFSRSBHRU5FUkFUSU5HIElOU1VGRklDSUVOVCBDQVNIIFRPIENPVkVSIENBUEVYLCBTTyBSSVNJTkcgREVCVDwvcD4NCjxwPk1hbnkgc21hbGxlciBwbGF5ZXJzIDxzcGFuIGNoYW5uZWw9IiEiPmxpa2UgRW5xdWVzdCA8L3NwYW4+aGF2ZSBhbHNvIGNvcGVkIGJ5IHRha2luZyBvbiBpbmNyZWFzaW5nIGFtb3VudHMgb2YgZGVidC48L3A+DQo8cD5QcmVtaWVyIE9pbCwgZm9yIGV4YW1wbGUsIGhhZCAkNDBtIG9mIG5ldCBjYXNoIGluIHRoZSBmaXJzdCBxdWFydGVyIG9mIDIwMDkuIEl0cyBsYXN0IHJlc3VsdHMgc2hvd2VkIHRoYXQgdGhpcyBoYXMgbm93IGJlY29tZSAkMi4xYm4gd29ydGggb2YgbmV0IGRlYnQsIGFuZCBhbmFseXN0cyBzYXkgdGhpcyB3aWxsIGtlZXAgcmlzaW5nIHVudGlsIGF0IGxlYXN0IHRoZSBlbmQgb2YgdGhlIHllYXIuIE5FVCBERUJUIFRPIEVCSVREQSBNVUxUSVBMRTwvcD4NCjxwPk92ZXIgdGhlIHNhbWUgcGVyaW9kLCB0aGUgY29tcGFueeKAmXMgb2lsIHJlc2VydmVzIGhhdmUgZmFsbGVuIHNsaWdodGx5IGZyb20gMjI4bSBiYXJyZWxzIG9mIG9pbCBlcXVpdmFsZW50IHRvIDIyM20uIFRoZSBleHRyYSBkZWJ0IGhhcyBnb25lIHRvd2FyZHMgbWFpbnRhaW5pbmcgc3RvY2tzPz8/PyByYXRoZXIgdGhhbiBncm93aW5nIHRoZW0uPC9wPg0KPHA+V2hpbGUgbWFueSBvZiB0aGVzZSBzbWFsbGVyIHBsYXllcnMgYXJlIGhpZ2hseSBpbmRlYnRlZCwgc2V2ZXJhbCBhcmUgYWxzbyB3b3JraW5nIG9uIGNvbnNpZGVyYWJsZSBuZXcgZmluZHMsIHN1Y2ggYXMgUHJlbWllcuKAmXMgU29sYW4gZmllbGQgd2VzdCBvZiBTaGV0bGFuZC4gTW9zdCBjb21wYW5pZXMgaGF2ZSBtYW5hZ2VkIHRvIHJlbmVnb3RpYXRlIHRoZWlyIGRlYnQgaW4gcmVjZW50IG1vbnRocywgbWVhbmluZyBiYW5raW5nIGNvdmVuYW50cyBhcmUgdW5saWtlbHkgdG8gYmUgYnJlYWNoZWQgaW4gdGhlIGltbWVkaWF0ZSB0ZXJtLiA8L3A+DQo8cD5BbmQgd2l0aCB0YXggYnJlYWtzIHJlY2VudGx5IG9mZmVyZWQgYnkgdGhlIFRyZWFzdXJ5LCBhbnkgaW5jcmVhc2UgaW4gcmV2ZW51ZSBpcyBsaWtlbHkgdG8gbGVhZCB0byBhIHNpbWlsYXIgaW5jcmVhc2VzIGluIHByb2ZpdHMuPC9wPg0KPHA+TWFyayBXaWxzb24sIGFuIGFuYWx5c3QgYXQgSmVmZmVyaWVzIGJhbmssIHNhaWQ6IOKAnFJpc2luZyBuZXQgZGVidCBpcyB0YWtpbmcgdmFsdWUgYXdheSBmcm9tIGVxdWl0eSBob2xkZXJzIGJ1dCBzdG9ja3MgaGF2ZSBhbiB1cHNpZGUgaWYgdGhleSBjYW4gZGVsaXZlciB3aGF0IGlzIG9uIHRoZWlyIGJhbGFuY2Ugc2hlZXQuIFdIQVQgSVMgSEUgUkVGRVJSSU5HIFRPIE9OIEJBTEFOQ0UgU0hFRVQ/4oCdPC9wPg0KPHA+PC9wPg0KPHA+QnV0IGlmIHRoZSBuZXcgZmluZHMgZGlzYXBwb2ludCwgb3IgaWYgdGhlIG9pbCBwcmljZSByZW1haW5zIHRvbyBsb3cgdG8gY292ZXIgY29zdHMgd2hlbiBvaWwgc3RhcnRzIGNvbWluZyBvdXQgb2YgdGhlIGdyb3VuZCwgdGhleSBjb3VsZCBmaW5kIHRoZW1zZWx2ZXMgZmluYW5jaWFsbHkgZGlzdHJlc3NlZC4gSU4gV0hBVCBXQVkgV0FTIEZBSVJGSUVMRCBESVNUUkVTU0VEPzwvcD4NCjxwPk9uZSBjb21wYW55IHRoYXQgaGFzIGdvbmUgdGhyb3VnaCB0aGlzIHByb2Nlc3MgaXMgRmFpcmZpZWxkLCB3aGljaCBoYXMgbWFkZSB0aGUgZGVjaXNpb24gdG8gYWJhbmRvbiBpdHMgRHVubGluIEFscGhhIHBsYXRmb3JtIGFuZCB0dXJuIGl0c2VsZiBpbnRvIGEgZGVjb21taXNzaW9uaW5nIHNwZWNpYWxpc3QgaW5zdGVhZC48L3A+DQo8cCBjaGFubmVsPSIhIj5UaGlzIGNvdWxkIGJlIGEgc291bmQgYnVzaW5lc3MgZGVjaXNpb246PC9wPg0KPHA+V2hhdGV2ZXIgaGFwcGVucywgbm9ib2R5IGludm9sdmVkIGluIE5vcnRoIFNlYSBvaWwgdGhpbmtzIGl0IHdpbGwgbG9vayB0aGUgc2FtZSBieSB0aGUgZW5kIG9mIHRoZSBkZWNhZGUuIDwvcD4NCjxwPk9uZSBzZW5pb3IgZXhlY3V0aXZlIGZyb20gYSBtYWpvciBvaWwgY29tcGFueSBzYWlkOiDigJxMb3RzIG9mIGNvbXBhbmllcyBkbyBub3QgcmVhbGlzZSBpdCB5ZXQsIGJ1dCB0aGlzIGlzIHRoZSBiZWdpbm5pbmcgb2YgdGhlIGVuZC7igJ08L3A+DQo8L2JvZHk+DQo8L3RleHQ+DQo8L3N0b3J5Pg0KPC9kb2M+DQo="
const attributesMarkedDeletedTrue = "<ObjectMetadata><OutputChannels><DIFTcom><DIFTcomMarkDeleted>True</DIFTcomMarkDeleted></DIFTcom></OutputChannels></ObjectMetadata>"
const attributesMarkedDeletedFalse = "<ObjectMetadata><OutputChannels><DIFTcom><DIFTcomMarkDeleted>False</DIFTcomMarkDeleted></DIFTcom></OutputChannels></ObjectMetadata>"

const validFileTypeAttributes = "<?xml version=\"1.0\" encoding=\"UTF-8\"?><!DOCTYPE ObjectMetadata SYSTEM \"/SysConfig/Classify/FTStories/classify.dtd\"><ObjectMetadata><EditorialDisplayIndexing><DIBylineCopy>Kiran Stacey, Energy Correspondent</DIBylineCopy></EditorialDisplayIndexing><OutputChannels><DIFTcom><DIFTcomWebType>story</DIFTcomWebType></DIFTcom></OutputChannels><EditorialNotes><Sources><Source><SourceCode>FT</SourceCode></Source></Sources><Language>English</Language><Author>staceyk</Author><ObjectLocation>/Users/staceyk/North Sea companies analysis CO 15.xml</ObjectLocation></EditorialNotes><DataFactoryIndexing><ADRIS_MetaData><IndexSuccess>yes</IndexSuccess><StartTime>Wed Oct 21 10:11:53 GMT 2015</StartTime><EndTime>Wed Oct 21 10:11:57 GMT 2015</EndTime></ADRIS_MetaData></DataFactoryIndexing></ObjectMetadata>"

const systemAttributesWebChannel = "<props><productInfo><name>FTcom</name><issueDate>20150915</issueDate></productInfo><workFolder>/FT/WorldNews</workFolder><subFolder>UKNews</subFolder></props>"
const contentWithHeadline = "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4NCjwhRE9DVFlQRSBkb2MgU1lTVEVNICIvU3lzQ29uZmlnL1J1bGVzL2Z0cHNpLmR0ZCI+DQo8P0VNLWR0ZEV4dCAvU3lzQ29uZmlnL1J1bGVzL2Z0cHNpL2Z0cHNpLmR0eD8+DQo8P0VNLXRlbXBsYXRlTmFtZSAvU3lzQ29uZmlnL1RlbXBsYXRlcy9GVC9CYXNlLVN0b3J5LnhtbD8+DQo8P3htbC1mb3JtVGVtcGxhdGUgL1N5c0NvbmZpZy9UZW1wbGF0ZXMvRlQvQmFzZS1TdG9yeS54cHQ/Pg0KPD94bWwtc3R5bGVzaGVldCB0eXBlPSJ0ZXh0L2NzcyIgaHJlZj0iL1N5c0NvbmZpZy9SdWxlcy9mdHBzaS9GVC9tYWlucmVwLmNzcyI/Pg0KPGRvYyB4bWw6bGFuZz0iZW4tdWsiPjxsZWFkIGlkPSJVMzIwMDExNzk2MDc0NzhGQkUiPjxsZWFkLWhlYWRsaW5lIGlkPSJVMzIwMDExNzk2MDc0NzhhMkQiPjxuaWQtdGl0bGU+PGxuPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgbmV3cyBpbiBkZXB0aCB0aXRsZSBoZXJlXT8+DQo8L2xuPg0KPC9uaWQtdGl0bGU+DQo8aW4tZGVwdGgtbmF2LXRpdGxlPjxsbj48P0VNLWR1bW15VGV4dCBbSW5zZXJ0IGluIGRlcHRoIG5hdiB0aXRsZSBoZXJlXT8+DQo8L2xuPg0KPC9pbi1kZXB0aC1uYXYtdGl0bGU+DQo8aGVhZGxpbmUgaWQ9IlUzMjAwMTE3OTYwNzQ3OHNtQyI+PGxuPlRoaXMgaXMgYSBoZWFkbGluZQ0KPC9sbj4NCjwvaGVhZGxpbmU+DQo8c2t5Ym94LWhlYWRsaW5lIGlkPSJVMzIwMDExNzk2MDc0Nzg1a0YiPjxsbj48P0VNLWR1bW15VGV4dCBbU2t5Ym94IGhlYWRsaW5lIGhlcmVdPz4NCjwvbG4+DQo8L3NreWJveC1oZWFkbGluZT4NCjx0cmlwbGV0LWhlYWRsaW5lPjxsbj48P0VNLWR1bW15VGV4dCBbVHJpcGxldCBoZWFkbGluZSBoZXJlXT8+DQo8L2xuPg0KPC90cmlwbGV0LWhlYWRsaW5lPg0KPHByb21vYm94LXRpdGxlPjxsbj48P0VNLWR1bW15VGV4dCBbUHJvbW9ib3ggdGl0bGUgaGVyZV0/Pg0KPC9sbj4NCjwvcHJvbW9ib3gtdGl0bGU+DQo8cHJvbW9ib3gtaGVhZGxpbmU+PGxuPjw/RU0tZHVtbXlUZXh0IFtQcm9tb2JveCBoZWFkbGluZSBoZXJlXT8+DQo8L2xuPg0KPC9wcm9tb2JveC1oZWFkbGluZT4NCjxlZGl0b3ItY2hvaWNlLWhlYWRsaW5lIGlkPSJVMzIwMDExNzk2MDc0NzhGVkUiPjxsbj48P0VNLWR1bW15VGV4dCBbU3RvcnkgcGFja2FnZSBoZWFkbGluZV0/Pg0KPC9sbj4NCjwvZWRpdG9yLWNob2ljZS1oZWFkbGluZT4NCjxuYXYtY29sbGVjdGlvbi1oZWFkbGluZT48bG4+PD9FTS1kdW1teVRleHQgW05hdiBjb2xsZWN0aW9uIGhlYWRsaW5lXT8+DQo8L2xuPg0KPC9uYXYtY29sbGVjdGlvbi1oZWFkbGluZT4NCjxpbi1kZXB0aC1uYXYtaGVhZGxpbmU+PGxuPjw/RU0tZHVtbXlUZXh0IFtJbiBkZXB0aCBuYXYgaGVhZGxpbmVdPz4NCjwvbG4+DQo8L2luLWRlcHRoLW5hdi1oZWFkbGluZT4NCjwvbGVhZC1oZWFkbGluZT4NCjx3ZWItaW5kZXgtaGVhZGxpbmUgaWQ9IlUzMjAwMTE3OTYwNzQ3OHRQRyI+PGxuPjw/RU0tZHVtbXlUZXh0IFtTaG9ydCBoZWFkbGluZV0/Pg0KPC9sbj4NCjwvd2ViLWluZGV4LWhlYWRsaW5lPg0KPHdlYi1zdGFuZC1maXJzdCBpZD0iVTMyMDAxMTc5NjA3NDc4bmVFIj48cD48P0VNLWR1bW15VGV4dCBbTG9uZyBzdGFuZGZpcnN0XT8+DQo8L3A+DQo8L3dlYi1zdGFuZC1maXJzdD4NCjx3ZWItc3ViaGVhZCBpZD0iVTMyMDAxMTc5NjA3NDc4dGpHIj48cD48P0VNLWR1bW15VGV4dCBbU2hvcnQgc3RhbmRmaXJzdF0/Pg0KPC9wPg0KPC93ZWItc3ViaGVhZD4NCjxlZGl0b3ItY2hvaWNlIGlkPSJVMzIwMDExNzk2MDc0NzhRWUUiPjw/RU0tZHVtbXlUZXh0IFtTdG9yeSBwYWNrYWdlIGh5cGVybGlua10/Pg0KPC9lZGl0b3ItY2hvaWNlPg0KPGxlYWQtdGV4dCBpZD0iVTMyMDAxMTc5NjA3NDc4T0VEIj48bGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgbGVhZCBib2R5IHRleHQgaGVyZSAtIG1pbiAxMzAgY2hhcnMsIG1heCAxNTAgY2hhcnNdPz4NCjwvcD4NCjwvbGVhZC1ib2R5Pg0KPHRyaXBsZXQtbGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgdHJpcGxldCBsZWFkIGJvZHkgYm9keSBoZXJlXT8+DQo8L3A+DQo8L3RyaXBsZXQtbGVhZC1ib2R5Pg0KPGNvbHVtbmlzdC1sZWFkLWJvZHk+PHA+PD9FTS1kdW1teVRleHQgW0luc2VydCBjb2x1bW5pc3QgbGVhZCBib2R5IGJvZHkgaGVyZV0/Pg0KPC9wPg0KPC9jb2x1bW5pc3QtbGVhZC1ib2R5Pg0KPHNob3J0LWJvZHk+PHA+PD9FTS1kdW1teVRleHQgW0luc2VydCBzaG9ydCBib2R5IGJvZHkgaGVyZV0/Pg0KPC9wPg0KPC9zaG9ydC1ib2R5Pg0KPHNreWJveC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgc2t5Ym94IGJvZHkgaGVyZV0/Pg0KPC9wPg0KPC9za3lib3gtYm9keT4NCjxwcm9tb2JveC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgcHJvbW9ib3ggYm9keSBoZXJlXT8+DQo8L3A+DQo8L3Byb21vYm94LWJvZHk+DQo8dHJpcGxldC1zaG9ydC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgdHJpcGxldCBzaG9ydCBib2R5IGhlcmVdPz4NCjwvcD4NCjwvdHJpcGxldC1zaG9ydC1ib2R5Pg0KPGVkaXRvci1jaG9pY2Utc2hvcnQtbGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgZWRpdG9ycyBjaG9pY2Ugc2hvcnQgbGVhZCBib2R5IGhlcmVdPz4NCjwvcD4NCjwvZWRpdG9yLWNob2ljZS1zaG9ydC1sZWFkLWJvZHk+DQo8bmF2LWNvbGxlY3Rpb24tc2hvcnQtbGVhZC1ib2R5PjxwPjw/RU0tZHVtbXlUZXh0IFtJbnNlcnQgbmF2IGNvbGxlY3Rpb24gc2hvcnQgbGVhZCBib2R5IGhlcmVdPz4NCjwvcD4NCjwvbmF2LWNvbGxlY3Rpb24tc2hvcnQtbGVhZC1ib2R5Pg0KPC9sZWFkLXRleHQ+DQo8bGVhZC1pbWFnZXMgaWQ9IlUzMjAwMTE3OTYwNzQ3OEFkQiI+PHdlYi1tYXN0ZXIgaWQ9IlUzMjAwMTE3OTYwNzQ3OFpSRiIvPg0KPHdlYi1za3lib3gtcGljdHVyZS8+DQo8d2ViLWFsdC1waWN0dXJlLz4NCjx3ZWItcG9wdXAtcHJldmlldyB3aWR0aD0iMTY3IiBoZWlnaHQ9Ijk2Ii8+DQo8d2ViLXBvcHVwLz4NCjwvbGVhZC1pbWFnZXM+DQo8aW50ZXJhY3RpdmUtY2hhcnQ+PD9FTS1kdW1teVRleHQgW0ludGVyYWN0aXZlLWNoYXJ0IGxpbmtdPz4NCjwvaW50ZXJhY3RpdmUtY2hhcnQ+DQo8L2xlYWQ+DQo8c3Rvcnk+PGhlYWRibG9jayBpZD0iVTMyMDAxMTc5NjA3NDc4d21IIj48aGVhZGxpbmUgaWQ9IlUzMjAwMTE3OTYwNzQ3OFgwSCI+PGxuPjw/RU0tZHVtbXlUZXh0IFtIZWFkbGluZV0/Pg0KPC9sbj4NCjwvaGVhZGxpbmU+DQo8L2hlYWRibG9jaz4NCjx0ZXh0IGlkPSJVMzIwMDExNzk2MDc0Nzh3WEQiPjxieWxpbmU+PGF1dGhvci1uYW1lPktpcmFuIFN0YWNleSwgRW5lcmd5IENvcnJlc3BvbmRlbnQ8L2F1dGhvci1uYW1lPg0KPC9ieWxpbmU+DQo8Ym9keT48cD5DT1NUIENPTVBBUklTSU9OIFdJVEggT1RIRVIgUkVHSU9OUzwvcD4NCjxwPjwvcD4NCjxwPlRvdmUgU3R1aHIgU2pvYmxvbSBzdG9vZCBpbiBmcm9udCBvZiAzMDAgb2lsIGluZHVzdHJ5IGluc2lkZXJzIGluIEFiZXJkZWVuIGxhc3Qgd2Vlaywgc2xpcHBlZCBpbnRvIHRoZSBsb2NhbCBTY290dGlzaCBkaWFsZWN0LCBhbmQgZ2F2ZSB0aGVtIG9uZSBjbGVhciBtZXNzYWdlOiDigJxLZWVwIHlvdXIgaGVpZC7igJ08L3A+DQo8cD5UaGUgTm9yd2VnaWFuIFN0YXRvaWwgZXhlY3V0aXZlIHdhcyBhZGRyZXNzaW5nIHRoZSBiaWVubmlhbCBPZmZzaG9yZSBFdXJvcGUgY29uZmVyZW5jZSwgZG9taW5hdGVkIHRoaXMgeWVhciBieSB0YWxrIG9mIHRoZSBmYWxsaW5nIG9pbCBwcmljZSBhbmQgdGhlIDxhIGhyZWY9Imh0dHA6Ly93d3cuZnQuY29tL2Ntcy9zLzAvNGYwNDQyZTAtYmNkMi0xMWU0LWE5MTctMDAxNDRmZWFiN2RlLmh0bWwjYXh6ejNsbllyTDN3YiIgdGl0bGU9Ik5vcnRoIFNlYSBvaWw6IFRoYXQgc2lua2luZyBmZWVsaW5nIC0gRlQiPmZ1dHVyZSBvZiBOb3J0aCBTZWEgcHJvZHVjdGlvbjwvYT4uIDwvcD4NCjxwPk92ZXIgdGhlIHBhc3QgMTQgbW9udGhzLCB0aGUgb2lsIHByaWNlIGhhcyBkcm9wcGVkIGZyb20gYWJvdXQgJDExNSBwZXIgYmFycmVsIHRvIGFyb3VuZCAkNTAuIE5vd2hlcmUgaGFzIHRoaXMgYmVlbiBmZWx0IG1vcmUga2Vlbmx5IHRoYW4gaW4gdGhlIE5vcnRoIFNlYSwgRk9SIERFQ0FERVMgYSBwb3dlcmhvdXNlIG9mIHRoZSBCcml0aXNoIGVjb25vbXksIGJ1dCBOT1cgYW4gYXJlYSB3aGVyZSBvaWwgZmllbGRzIGFyZSBnZXR0aW5nIG9sZGVyLCBuZXcgZGlzY292ZXJpZXMgcmFyZXIgYW5kIGNvc3RzIHN0ZWVwZXIuIE5vIHdvbmRlciBNcyBTdHVociBTam9ibG9tIGlzIHRyeWluZyB0byBzdG9wIGhlciBjb2xsZWFndWVzIGZyb20gcGFuaWNraW5nLjwvcD4NCjxwPkJ5IG5lYXJseSBhbnkgbWV0cmljLCBkb2luZyBidXNpbmVzcyBpbiB0aGUgTm9ydGggU2VhIGlzIG1vcmUgZGlmZmljdWx0IHRoYW4gYXQgYW55IG90aGVyIHBlcmlvZCBpbiBpdHMgNTAteWVhciBoaXN0b3J5LiBXaGlsZSB0aGUgb2lsIHByaWNlIGhhcyBiZWVuIGxvd2VyIOKAlCBpdCB0b3VjaGVkICQxMCBhIGJhcnJlbCBpbiAxOTg2IOKAlCBzdWNoIGEgcmFwaWQgc2x1bXAgaGFzIG5ldmVyIGJlZW4gc2VlbiB3aGVuIGNvc3RzIHdlcmUgc28gaGlnaCBhbmQgd2l0aCBvaWwgcHJvZHVjdGlvbiBhbHJlYWR5IGluIGRlY2xpbmUuPC9wPg0KPHA+SVMgUFJPRFVDVElPTiBJTiBERUNMSU5FIEJFQ0FVU0UgTk9SVEggU0VBIElTIFJVTk5JTkcgRFJZPyBCVVQgRE9FUyBUSElTIERPV05UVVJOIFJJU0sgTUFTU0lWRSBBQ0NFTEVSQVRJT04gT0YgVEhBVCBERUNMSU5FIEJFQ0FVU0UgRVhQTE9SQVRJT04vREVWRUxPUE1FTlQgSVMgQkVJTkcgQ1VSVEFJTEVEPC9wPg0KPHA+T25lIGV4ZWN1dGl2ZSBmcm9tIGFuIG9pbCBtYWpvciBzYXlzOiDigJxXZSBoYXZlIG5ldmVyIGJlZm9yZSBleHBlcmllbmNlZCBhbnl0aGluZyBsaWtlIHRoaXMgYmVmb3JlIOKAlCBldmVyeXRoaW5nIGlzIGhhcHBlbmluZyBhdCB0aGUgc2FtZSB0aW1lLuKAnTwvcD4NCjxwPkFjY29yZGluZyB0byBhbmFseXNpcyBieSBFd2FuIE11cnJheSBvZiB0aGUgY29ycG9yYXRlIGhlYWx0aCBtb25pdG9yIENvbXBhbnkgV2F0Y2gsIG91dCBvZiB0aGUgMTI2IG9pbCBleHBsb3JhdGlvbiBhbmQgcHJvZHVjdGlvbiBjb21wYW5pZXMgbGlzdGVkIGluIExvbmRvbiwgNzcgcGVyIGNlbnQgYXJlIG5vdyBsb3NzLW1ha2luZy4gQVJFIEFMTCBUSEVTRSBJTiBUSEUgTk9SVEggU0VBPz8/IFRvdGFsIGxvc3NlcyBzdGFuZCBhdCDCozYuNmJuLjwvcD4NCjxwPjxhIGhyZWY9Imh0dHA6Ly9vaWxhbmRnYXN1ay5jby51ay9lY29ub21pYy1yZXBvcnQtMjAxNS5jZm0iIHRpdGxlPSJPaWwgJmFtcDsgR2FzIEVjb25vbWljIFJlcG9ydCAyMDE1Ij5GaWd1cmVzIHJlbGVhc2VkIGxhc3Qgd2VlayBieSBPaWwgJmFtcDsgR2FzIFVLPC9hPiwgdGhlIGluZHVzdHJ5IGJvZHksIHNob3cgdGhhdCBmb3IgdGhlIGZpcnN0IHRpbWUgc2luY2UgMTk3Nywgd2hlbiBOb3J0aCBTZWEgb2lsIHdhcyBzdGlsbCB5b3VuZywgaXQgbm93IGNvc3RzIG1vcmUgdG8gcHVtcCBvaWwgb3V0IG9mIHRoZSBzZWFiZWQgdGhhbiBpdCBnZW5lcmF0ZXMgaW4gcG9zdC10YXggcmV2ZW51ZT8/PyBQUk9GSVQ/Pz8uIDwvcD4NCjxwPjxhIGhyZWY9Imh0dHA6Ly9wdWJsaWMud29vZG1hYy5jb20vcHVibGljL21lZGlhLWNlbnRyZS8xMjUyOTI1NCIgdGl0bGU9IkxvdyBvaWwgcHJpY2UgYWNjZWxlcmF0aW5nIGRlY29tbWlzc2lvbmluZyBpbiB0aGUgVUtDUyAtIFdvb2RNYWMiPmFuYWx5c2lzIHB1Ymxpc2hlZCBsYXN0IHdlZWsgYnkgV29vZCBNYWNrZW56aWU8L2E+c3VnZ2VzdHMgMTQwIGZpZWxkcyBtYXkgY2xvc2UgaW4gdGhlIG5leHQgZml2ZSB5ZWFycywgZXZlbiBpZiB0aGUgb2lsIHByaWNlIHJldHVybnMgdG8gJDg1IGEgYmFycmVsLjwvcD4NCjxwPkRpZmZlcmVudCBjb21wYW5pZXMgb3BlcmF0aW5nIGluIHRoZSBOb3J0aCBTZWEgaGF2ZSByZXNwb25kZWQgdG8gdGhpcyBpbiBkaWZmZXJlbnQgd2F5cy48L3A+DQo8cD5PaWwgbWFqb3JzIHRlbmQgdG8gaGF2ZSBhbiBleGl0IHJvdXRlLiBUaGV5IGhhdmUgY3V0IGpvYnMg4oCUIDUsNTAwIGhhdmUgYmVlbiBsb3N0IHNvIGZhciwgd2l0aCBleGVjdXRpdmVzIHdhcm5pbmcgb2YgYW5vdGhlciAxMCwwMDAgaW4gdGhlIGNvbWluZyB5ZWFycyDigJQgYW5kIGFyZSBub3cgdHJ5aW5nIHRvIFJFRFVDRSBPVEhFUiBDT1NUUyBTVUNIIEFTPz8/IGN1dCBjb3N0cy4gQnV0IGlmIGFsbCBlbHNlIGZhaWxzLCB0aGV5IGNhbiBzaW1wbHkgZXhpdCB0aGUgcmVnaW9uIGFuZCBmb2N1cyB0aGVpciBlZmZvcnRzIG9uIGNoZWFwZXIgZmllbGRzIGVsc2V3aGVyZSBpbiB0aGUgd29ybGQuPC9wPg0KPHA+RnJhbmNl4oCZcyBUb3RhbCBsYXN0IG1vbnRoIGFncmVlZCB0byBzZWxsIE5vcnRoIFNlYSA8YSBocmVmPSJodHRwOi8vd3d3LmZ0LmNvbS9jbXMvcy8wLzg5MWQ2ZTU4LTRjMTktMTFlNS1iNTU4LThhOTcyMjk3NzE4OS5odG1sI2F4enozbG5Zckwzd2IiIHRpdGxlPSJGcmVuY2ggb2lsIG1ham9yIFRvdGFsIHRvIHNlbGwgJDkwMG0gb2YgTm9ydGggU2VhIGFzc2V0cyAtIEZUIj5nYXMgdGVybWluYWxzIGFuZCBwaXBlbGluZXM8L2E+IGluIGEgJDkwMG0gZGVhbCwgd2hpbGUgRW9uLCB0aGUgR2VybWFuIHV0aWxpdHkgY29tcGFueSwgaXMgbG9va2luZyBmb3IgYnV5ZXJzIGZvciBzb21lIG9mIGl0cyBhc3NldHMgaW4gdGhlIHJlZ2lvbi48L3A+DQo8cD5TbWFsbGVyIGNvbXBhbmllcyBob3dldmVyIHRlbmQgbm90IHRvIGhhdmUgdGhhdCBvcHRpb24uIEZvciBzb21lLCB0aGUgb2lsIHByaWNlIHBsdW5nZSBtZWFucyBzcXVlZXppbmcgY29zdHMgYXMgbXVjaCBhcyBwb3NzaWJsZSBpbiBhbiBlZmZvcnQgdG8gc3RheSBhZmxvYXQgdW50aWwgaXQgcmVib3VuZHMuPC9wPg0KPHA+RE8gQ0FQRVggQ1VUIEZJUlNUIEJZIEUgTlFVRVNUPC9wPg0KPHA+VEhFTiBFWFBMQUlOIEZPQ1VTIE9OIEVYVFJBQ1RJTkcgTU9SRSBGUk9NIEZFVyBGSUVMRFMgSVQgSVMgRk9DVVNJTkcgT04sIEFORCBDVVRUSU5HIE9QRVg8L3A+DQo8cD48L3A+DQo8cD5FbnF1ZXN0LCBmb3IgZXhhbXBsZSwgaGFzIG1hZGUgYSB2aXJ0dWUgZnJvbSBnZXR0aW5nIG1vcmUgb2lsIG91dCBvZiBtYXR1cmUgZmllbGRzIHRoYW4gbWFqb3JzIGNhbi4gSXRzIFRoaXN0bGUgb2lsIGZpZWxkIHdhcyBvbmNlIG93bmVkIGJ5IEJQLCBidXQgYXMgcHJvZHVjdGlvbiBkZWNsaW5lZCwgRW5xdWVzdCBzdGVwcGVkIGluLCBhbmQgYnkgbGFzdCB5ZWFyIHRoZSBjb21wYW55IG1hbmFnZWQgdG8gZXh0cmFjdCAzbSBiYXJyZWxzIGZyb20gaXQgZm9yIHRoZSBmaXJzdCB0aW1lIHNpbmNlIDE5OTcuPC9wPg0KPHA+T25lIHdheSB0byBkbyBzbz8/Pz8gaXMgdG8gY3V0IGNvc3RzIGFnZ3Jlc3NpdmVseS4gRW5xdWVzdCBoYXMgbG9va2VkIGF0IGV2ZXJ5dGhpbmcgaW5jbHVkaW5nIHJlbG9jYXRpbmcgaXRzIGhlbGljb3B0ZXIgc2VydmljZSBpbiBhbiBlZmZvcnQgdG8gc2F2ZSBtb25leS4gQW5kIG1vcmUgaXMgbGlrZWx5IHRvIGNvbWUuPC9wPg0KPHA+QW1qYWQgQnNlc2l1LCB0aGUgY29tcGFueeKAmXMgY2hpZWYgZXhlY3V0aXZlLCBzYWlkOiDigJxUaGUgaW5kdXN0cnkgd2VudCB0aHJvdWdoIGxhc3QgeWVhcuKAmXMgY3ljbGUgaW4gYSBsaXR0bGUgYml0IG9mIGEgdGltaWQgbWFubmVyLiBUaGVyZSB3YXMgYSBmZWVsaW5nIHRoYXQgbWF5YmUgdGhpbmdzIGNhbiBjb21lIGJhY2sgbGlrZSB0aGV5IGRpZCBpbiAyMDA4IHRvIDIwMDk/Pz8sIGJ1dCB0aGF0IGhhc27igJl0IGhhcHBlbmVkLuKAnTwvcD4NCjxwPkJ1dCBmb3IgbWFueSBzbWFsbGVyIGNvbXBhbmllcywgY3V0dGluZyBjb3N0cyBoYXMgYWxzbyBpbnZvbHZlZCBjdXR0aW5nIGV4cGxvcmF0aW9uIGFuZCBkZXZlbG9wbWVudC4gT3V0IG9mIHRoZSAxMiBmaWVsZHMgaXQgb3ducyBpbiB3aGljaCBvaWwgaGFzIGJlZW4gZGlzY292ZXJlZCwgRW5xdWVzdCBpcyBjdXJyZW50bHkgZGV2ZWxvcGluZyBvbmx5IHR3by4gVGhpcyBpcyBhIHRyZW5kIG92ZXIgdGhlIGluZHVzdHJ5IGFzIGEgd2hvbGU6IGluIDIwMDggdGhlIGluZHVzdHJ5IGRyaWxsZWQgNDAgZXhwbG9yYXRpb24gd2VsbHMgaW4gdGhlIE5vcnRoIFNlYS4gTGFzdCB5ZWFyLCB0aGF0IGZpZ3VyZSB3YXMganVzdCAxNC48L3A+DQo8cD48L3A+DQo8cD5CVVQgU09NRSBDT01QQU5JRVMgQVJFIFBST0NFRURJTkcgV0lUSCBCSUcgUFJPSkVDVFM6IFNVQ0ggQVMgUFJFTUlFUiBBVCBTT0xBTjwvcD4NCjxwPklUIE1FQU5TIFNPTUUgQ09NUEFOSUVTIEFSRSBHRU5FUkFUSU5HIElOU1VGRklDSUVOVCBDQVNIIFRPIENPVkVSIENBUEVYLCBTTyBSSVNJTkcgREVCVDwvcD4NCjxwPk1hbnkgc21hbGxlciBwbGF5ZXJzIDxzcGFuIGNoYW5uZWw9IiEiPmxpa2UgRW5xdWVzdCA8L3NwYW4+aGF2ZSBhbHNvIGNvcGVkIGJ5IHRha2luZyBvbiBpbmNyZWFzaW5nIGFtb3VudHMgb2YgZGVidC48L3A+DQo8cD5QcmVtaWVyIE9pbCwgZm9yIGV4YW1wbGUsIGhhZCAkNDBtIG9mIG5ldCBjYXNoIGluIHRoZSBmaXJzdCBxdWFydGVyIG9mIDIwMDkuIEl0cyBsYXN0IHJlc3VsdHMgc2hvd2VkIHRoYXQgdGhpcyBoYXMgbm93IGJlY29tZSAkMi4xYm4gd29ydGggb2YgbmV0IGRlYnQsIGFuZCBhbmFseXN0cyBzYXkgdGhpcyB3aWxsIGtlZXAgcmlzaW5nIHVudGlsIGF0IGxlYXN0IHRoZSBlbmQgb2YgdGhlIHllYXIuIE5FVCBERUJUIFRPIEVCSVREQSBNVUxUSVBMRTwvcD4NCjxwPk92ZXIgdGhlIHNhbWUgcGVyaW9kLCB0aGUgY29tcGFueeKAmXMgb2lsIHJlc2VydmVzIGhhdmUgZmFsbGVuIHNsaWdodGx5IGZyb20gMjI4bSBiYXJyZWxzIG9mIG9pbCBlcXVpdmFsZW50IHRvIDIyM20uIFRoZSBleHRyYSBkZWJ0IGhhcyBnb25lIHRvd2FyZHMgbWFpbnRhaW5pbmcgc3RvY2tzPz8/PyByYXRoZXIgdGhhbiBncm93aW5nIHRoZW0uPC9wPg0KPHA+V2hpbGUgbWFueSBvZiB0aGVzZSBzbWFsbGVyIHBsYXllcnMgYXJlIGhpZ2hseSBpbmRlYnRlZCwgc2V2ZXJhbCBhcmUgYWxzbyB3b3JraW5nIG9uIGNvbnNpZGVyYWJsZSBuZXcgZmluZHMsIHN1Y2ggYXMgUHJlbWllcuKAmXMgU29sYW4gZmllbGQgd2VzdCBvZiBTaGV0bGFuZC4gTW9zdCBjb21wYW5pZXMgaGF2ZSBtYW5hZ2VkIHRvIHJlbmVnb3RpYXRlIHRoZWlyIGRlYnQgaW4gcmVjZW50IG1vbnRocywgbWVhbmluZyBiYW5raW5nIGNvdmVuYW50cyBhcmUgdW5saWtlbHkgdG8gYmUgYnJlYWNoZWQgaW4gdGhlIGltbWVkaWF0ZSB0ZXJtLiA8L3A+DQo8cD5BbmQgd2l0aCB0YXggYnJlYWtzIHJlY2VudGx5IG9mZmVyZWQgYnkgdGhlIFRyZWFzdXJ5LCBhbnkgaW5jcmVhc2UgaW4gcmV2ZW51ZSBpcyBsaWtlbHkgdG8gbGVhZCB0byBhIHNpbWlsYXIgaW5jcmVhc2VzIGluIHByb2ZpdHMuPC9wPg0KPHA+TWFyayBXaWxzb24sIGFuIGFuYWx5c3QgYXQgSmVmZmVyaWVzIGJhbmssIHNhaWQ6IOKAnFJpc2luZyBuZXQgZGVidCBpcyB0YWtpbmcgdmFsdWUgYXdheSBmcm9tIGVxdWl0eSBob2xkZXJzIGJ1dCBzdG9ja3MgaGF2ZSBhbiB1cHNpZGUgaWYgdGhleSBjYW4gZGVsaXZlciB3aGF0IGlzIG9uIHRoZWlyIGJhbGFuY2Ugc2hlZXQuIFdIQVQgSVMgSEUgUkVGRVJSSU5HIFRPIE9OIEJBTEFOQ0UgU0hFRVQ/4oCdPC9wPg0KPHA+PC9wPg0KPHA+QnV0IGlmIHRoZSBuZXcgZmluZHMgZGlzYXBwb2ludCwgb3IgaWYgdGhlIG9pbCBwcmljZSByZW1haW5zIHRvbyBsb3cgdG8gY292ZXIgY29zdHMgd2hlbiBvaWwgc3RhcnRzIGNvbWluZyBvdXQgb2YgdGhlIGdyb3VuZCwgdGhleSBjb3VsZCBmaW5kIHRoZW1zZWx2ZXMgZmluYW5jaWFsbHkgZGlzdHJlc3NlZC4gSU4gV0hBVCBXQVkgV0FTIEZBSVJGSUVMRCBESVNUUkVTU0VEPzwvcD4NCjxwPk9uZSBjb21wYW55IHRoYXQgaGFzIGdvbmUgdGhyb3VnaCB0aGlzIHByb2Nlc3MgaXMgRmFpcmZpZWxkLCB3aGljaCBoYXMgbWFkZSB0aGUgZGVjaXNpb24gdG8gYWJhbmRvbiBpdHMgRHVubGluIEFscGhhIHBsYXRmb3JtIGFuZCB0dXJuIGl0c2VsZiBpbnRvIGEgZGVjb21taXNzaW9uaW5nIHNwZWNpYWxpc3QgaW5zdGVhZC48L3A+DQo8cCBjaGFubmVsPSIhIj5UaGlzIGNvdWxkIGJlIGEgc291bmQgYnVzaW5lc3MgZGVjaXNpb246PC9wPg0KPHA+V2hhdGV2ZXIgaGFwcGVucywgbm9ib2R5IGludm9sdmVkIGluIE5vcnRoIFNlYSBvaWwgdGhpbmtzIGl0IHdpbGwgbG9vayB0aGUgc2FtZSBieSB0aGUgZW5kIG9mIHRoZSBkZWNhZGUuIDwvcD4NCjxwPk9uZSBzZW5pb3IgZXhlY3V0aXZlIGZyb20gYSBtYWpvciBvaWwgY29tcGFueSBzYWlkOiDigJxMb3RzIG9mIGNvbXBhbmllcyBkbyBub3QgcmVhbGlzZSBpdCB5ZXQsIGJ1dCB0aGlzIGlzIHRoZSBiZWdpbm5pbmcgb2YgdGhlIGVuZC7igJ08L3A+DQo8L2JvZHk+DQo8L3RleHQ+DQo8L3N0b3J5Pg0KPC9kb2M+DQo="
