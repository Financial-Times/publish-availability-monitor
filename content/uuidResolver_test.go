package content

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResolveIdentifier_Ok(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("ContentQuery", "http://api.ft.com/system/FT-LABS-WP-1-24", "http://ftalphaville.ft.com/?p=2193913", "tid_1").Return(http.StatusMovedPermanently, "http://api.ft.com/content/5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", nil)

	resolver := NewHTTPUUIDResolver(mockClient, map[string]string{"ftalphaville.ft.com": "FT-LABS-WP-1-24"})
	uuid, err := resolver.ResolveIdentifier("http://ftalphaville.ft.com/?p=2193913", "2193913", "tid_1")

	assert.NoError(t, err, "Should resolve fine.")
	assert.Equal(t, "5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", uuid)
}

func TestResolveIdentifier_NotInMap(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("ContentQuery", "http://api.ft.com/system/FT-LABS-WP-1-24", "http://ftalphaville.ft.com/?p=2193913", "tid_1").Return(http.StatusMovedPermanently, "http://api.ft.com/content/5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", nil)

	resolver := NewHTTPUUIDResolver(mockClient, map[string]string{})
	_, err := resolver.ResolveIdentifier("http://ftalphaville.ft.com/?p=2193913", "2193913", "tid_1")

	assert.True(t, strings.Contains(err.Error(), "couldn't find authority in mapping table"))
}

func TestResolveIdentifier_InvalidUuid(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("ContentQuery", "http://api.ft.com/system/FT-LABS-WP-1-24", "http://ftalphaville.ft.com/?p=2193913", "tid_1").Return(http.StatusMovedPermanently, "http://api.ft.com/content/5414b08f-xxxxx", nil)

	resolver := NewHTTPUUIDResolver(mockClient, map[string]string{"ftalphaville.ft.com": "FT-LABS-WP-1-24"})
	_, err := resolver.ResolveIdentifier("http://ftalphaville.ft.com/?p=2193913", "2193913", "tid_1")

	assert.True(t, strings.Contains(err.Error(), "invalid uuid"))
}

func TestResolveIdentifier_InvalidLocation(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("ContentQuery", "http://api.ft.com/system/FT-LABS-WP-1-24", "http://ftalphaville.ft.com/?p=2193913", "tid_1").Return(http.StatusMovedPermanently, "wrong", nil)

	resolver := NewHTTPUUIDResolver(mockClient, map[string]string{"ftalphaville.ft.com": "FT-LABS-WP-1-24"})
	_, err := resolver.ResolveIdentifier("http://ftalphaville.ft.com/?p=2193913", "2193913", "tid_1")

	assert.True(t, strings.Contains(err.Error(), "invalid FT URI"))
}

func TestResolveIdentifier_NotFound(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("ContentQuery", "http://api.ft.com/system/FT-LABS-WP-1-24", "http://ftalphaville.ft.com/?p=2193913", "tid_1").Return(http.StatusNotFound, "", nil)

	resolver := NewHTTPUUIDResolver(mockClient, map[string]string{"ftalphaville.ft.com": "FT-LABS-WP-1-24"})
	_, err := resolver.ResolveIdentifier("http://ftalphaville.ft.com/?p=2193913", "2193913", "tid_1")

	assert.True(t, strings.Contains(err.Error(), "404"))
}

func TestResolveIdentifier_NetFail(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("ContentQuery", "http://api.ft.com/system/FT-LABS-WP-1-24", "http://ftalphaville.ft.com/?p=2193913", "tid_1").Return(-1, "", errors.New("Couldn't make HTTP call"))

	resolver := NewHTTPUUIDResolver(mockClient, map[string]string{"ftalphaville.ft.com": "FT-LABS-WP-1-24"})
	_, err := resolver.ResolveIdentifier("http://ftalphaville.ft.com/?p=2193913", "2193913", "tid_1")

	assert.Equal(t, "Couldn't make HTTP call", err.Error())
}

func TestResolveOriginalUUID_Ok(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("IsUUIDPresent", "5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", "tid_1").Return(true, nil)

	resolver := httpResolver{client: mockClient}
	uuid, err := resolver.ResolveOriginalUUID("5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", "tid_1")

	assert.NoError(t, err, "Should resolve fine.")
	assert.Equal(t, "5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", uuid)
}

func TestResolveOriginalUUID_InvalidUUID(t *testing.T) {
	resolver := httpResolver{}
	_, err := resolver.ResolveOriginalUUID("InvalidUUID", "tid_1")

	assert.Error(t, err, "couldn't resolve OriginalUUID=InvalidUUID for tid=tid_1 because it's not a valid UUID")
}

func TestResolveOriginalUUID_NotFound(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("IsUUIDPresent", "5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", "tid_1").Return(false, nil)

	resolver := httpResolver{client: mockClient}
	uuid, err := resolver.ResolveOriginalUUID("5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", "tid_1")

	assert.NoError(t, err, "Should resolve fine.")
	assert.Equal(t, "", uuid)
}

func TestResolveOriginalUUID_NetFail(t *testing.T) {
	mockClient := new(MockDocStoreClient)
	mockClient.On("IsUUIDPresent", "5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", "tid_1").Return(false, errors.New("Couldn't make HTTP call"))

	resolver := httpResolver{client: mockClient}
	_, err := resolver.ResolveOriginalUUID("5414b08f-5ae1-3bd6-9901-a9dd1bf9db03", "tid_1")

	assert.Equal(t, "couldn't resolve OriginalUUID=5414b08f-5ae1-3bd6-9901-a9dd1bf9db03 for tid=tid_1, error was: Couldn't make HTTP call", err.Error())
}

type MockDocStoreClient struct {
	mock.Mock
}

func (m *MockDocStoreClient) ContentQuery(authority string, identifier string, tid string) (status int, location string, err error) {
	args := m.Called(authority, identifier, tid)
	return args.Int(0), args.String(1), args.Error(2)
}

func (m *MockDocStoreClient) IsUUIDPresent(uuid, tid string) (isPresent bool, err error) {
	args := m.Called(uuid, tid)
	return args.Bool(0), args.Error(1)
}
