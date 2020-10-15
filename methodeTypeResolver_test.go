package main

import (
	"strings"
	"testing"

	"github.com/Financial-Times/publish-availability-monitor/v2/content"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const wordpressAttributesXML = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\\n" +
	"<!DOCTYPE ObjectMetadata SYSTEM \"/SysConfig/Classify/FTStories/classify.dtd\">" +
	"<ObjectMetadata>" +
	"<EditorialNotes><OriginalUUID>{uuid}</OriginalUUID></EditorialNotes>" +
	"<WiresIndexing>" +
	"<category>blog</category>" +
	"<serviceid>http://ftalphaville.ft.com/?p=2194657</serviceid>" +
	"<ref_field>2194657</ref_field>" +
	"</WiresIndexing>" +
	"</ObjectMetadata>"

const notWordpressAttributesXML = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\\n" +
	"<!DOCTYPE ObjectMetadata SYSTEM \"/SysConfig/Classify/FTStories/classify.dtd\">" +
	"<ObjectMetadata>" +
	"<WiresIndexing>" +
	"<category>no-blog-category</category>" +
	"</WiresIndexing>" +
	"</ObjectMetadata>"

func TestResolveTypeAndUuid_NotCPH(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "FT",
		},
	}

	typeResolver := methodeTypeResolver{}

	resultType, resultUUID, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")

	assert.NoError(t, err, "Normal methode article shouldn't throw error on type and uuid resolve.")
	assert.Equal(t, "EOM::CompoundStory", resultType)
	assert.Equal(t, "e28b12f7-9796-3331-b030-05082f0b8157", resultUUID)
}

func TestResolveTypeAndUuid_InternalCPH_WithoutOriginalUUID(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "ContentPlaceholder",
		},
		Attributes: strings.Replace(wordpressAttributesXML, "{uuid}", "", -1),
	}

	resolverMock := new(MockUUIDResolver)
	resolverMock.On("ResolveIdentifier", "http://ftalphaville.ft.com/?p=2194657", "2194657", "tid_0123wxyz").Return("f3dbacdf-9796-3331-b030-05082f0b8157", nil)
	typeResolver := methodeTypeResolver{resolver: resolverMock}

	resultType, resultUUID, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")

	assert.NoError(t, err, "Internal CPH shouldn't throw error on type and uuid resolve.")
	assert.Equal(t, "EOM::CompoundStory_Internal_CPH", resultType)
	assert.Equal(t, "f3dbacdf-9796-3331-b030-05082f0b8157", resultUUID)
}

func TestResolveTypeAndUuid_InternalCPH_WithOriginalUUID(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "ContentPlaceholder",
		},
		Attributes: strings.Replace(wordpressAttributesXML, "{uuid}", "f3dbacdf-9796-3331-b030-05082f0b8157", -1),
	}

	resolverMock := new(MockUUIDResolver)
	resolverMock.On("ResolveOriginalUUID", "f3dbacdf-9796-3331-b030-05082f0b8157", "tid_0123wxyz").Return("f3dbacdf-9796-3331-b030-05082f0b8157", nil)
	typeResolver := methodeTypeResolver{resolver:resolverMock}

	resultType, resultUUID, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")

	assert.NoError(t, err, "Internal CPH shouldn't throw error on type and uuid resolve.")
	assert.Equal(t, "EOM::CompoundStory_Internal_CPH", resultType)
	assert.Equal(t, "f3dbacdf-9796-3331-b030-05082f0b8157", resultUUID)
}

func TestResolveTypeAndUuid_InternalCPH_WithOriginalUUID_ResolverError(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "ContentPlaceholder",
		},
		Attributes: strings.Replace(wordpressAttributesXML, "{uuid}", "f3dbacdf-9796-3331-b030-05082f0b8157", -1),
	}

	resolverMock := new(MockUUIDResolver)
	resolverMock.On("ResolveOriginalUUID", "f3dbacdf-9796-3331-b030-05082f0b8157", "tid_0123wxyz").Return("", errors.New("Resolver error"))
	typeResolver := methodeTypeResolver{resolver:resolverMock}

	_, _, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")

	assert.Error(t, err, "Resolver error")
}

func TestResolveTypeAndUuid_InternalCPH_WithOriginalUUID_MissingUUID(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "ContentPlaceholder",
		},
		Attributes: strings.Replace(wordpressAttributesXML, "{uuid}", "f3dbacdf-9796-3331-b030-05082f0b8157", -1),
	}

	resolverMock := new(MockUUIDResolver)
	resolverMock.On("ResolveOriginalUUID", "f3dbacdf-9796-3331-b030-05082f0b8157", "tid_0123wxyz").Return("", nil)
	typeResolver := methodeTypeResolver{resolver:resolverMock}

	_, _, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")

	assert.Error(t, err, "couldn't resolve CPH uuid for tid=tid_0123wxyz, OriginalUUID=f3dbacdf-9796-3331-b030-05082f0b8157 is not present in the database")
}

func TestResolveTypeAndUuid_ExternalCPH(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "ContentPlaceholder",
		},
		Attributes: notWordpressAttributesXML,
	}

	typeResolver := methodeTypeResolver{}

	resultType, resultUUID, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")

	assert.NoError(t, err, "Internal CPH shouldn't throw error on type and uuid resolve.")
	assert.Equal(t, "EOM::CompoundStory_External_CPH", resultType)
	assert.Equal(t, "e28b12f7-9796-3331-b030-05082f0b8157", resultUUID)
}

func TestResolveTypeAndUuid_InternalCPH_FailResolve_ThrowsError(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "ContentPlaceholder",
		},
		Attributes: strings.Replace(wordpressAttributesXML, "{uuid}", "", -1),
	}

	resolverMock := new(MockUUIDResolver)
	resolverMock.On("ResolveIdentifier", "http://ftalphaville.ft.com/?p=2194657", "2194657", "tid_0123wxyz").Return("", errors.New("Error calling UUID resolver"))
	typeResolver := methodeTypeResolver{resolver: resolverMock}

	_, _, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")
	assert.Error(t, err, "Internal CPH should throw error on failing to resolve UUID.")
}

func TestResolveTypeAndUuid_CPH_InvalidAttributes_ThrowsError(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "ContentPlaceholder",
		},
		Attributes: "some invalid attributes",
	}

	typeResolver := methodeTypeResolver{}

	_, _, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")
	assert.Error(t, err, "CPH should throw error on invalid attributes.")
}

func TestResolveTypeAndUuid_DynamicContent(t *testing.T) {
	eomFile := content.EomFile{
		UUID:        "e28b12f7-9796-3331-b030-05082f0b8157",
		ContentType: "EOM::CompoundStory",
		Source: content.Source{
			SourceCode: "DynamicContent",
		},
		Attributes: notWordpressAttributesXML,
	}

	typeResolver := methodeTypeResolver{}

	resultType, resultUUID, err := typeResolver.ResolveTypeAndUuid(eomFile, "tid_0123wxyz")

	assert.NoError(t, err, "Internal CPH shouldn't throw error on type and uuid resolve.")
	assert.Equal(t, "EOM::CompoundStory_DynamicContent", resultType)
	assert.Equal(t, "e28b12f7-9796-3331-b030-05082f0b8157", resultUUID)
}

type MockUUIDResolver struct {
	mock.Mock
}

func (m *MockUUIDResolver) ResolveIdentifier(serviceId, refField, tid string) (string, error) {
	args := m.Called(serviceId, refField, tid)
	return args.String(0), args.Error(1)
}

func (m *MockUUIDResolver) ResolveOriginalUUID(uuid, tid string) (string, error) {
	args := m.Called(uuid, tid)
	return args.String(0), args.Error(1)
}
