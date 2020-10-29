package content

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	authorityPrefix = "http://api.ft.com/system/"
	uuidPattern     = "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"
)

var uuidRegex = regexp.MustCompile(uuidPattern)

type UUIDResolver interface {
	ResolveIdentifier(serviceID, refField, tid string) (string, error)
	ResolveOriginalUUID(uuid, tid string) (string, error)
}

type httpResolver struct {
	brandMappings map[string]string
	client        DocStoreClient
}

func NewHTTPUUIDResolver(client DocStoreClient, brandMappings map[string]string) UUIDResolver {
	return &httpResolver{client: client, brandMappings: brandMappings}
}

func (r *httpResolver) ResolveIdentifier(serviceID, refField, tid string) (string, error) {
	mappingKey := strings.Split(serviceID, "?")[0]
	mappingKey = strings.Split(mappingKey, "#")[0]
	for key, value := range r.brandMappings {
		if strings.Contains(mappingKey, key) {
			authority := authorityPrefix + value
			identifierValue := strings.Split(serviceID, "://")[0] + "://" + key + "/?p=" + refField
			return r.resolveIdentifier(authority, identifierValue, tid)
		}
	}
	return "", fmt.Errorf("couldn't find authority in mapping table tid=%v serviceId=%v refField=%v", tid, serviceID, refField)
}

func (r *httpResolver) ResolveOriginalUUID(uuid, tid string) (string, error) {
	if !uuidRegex.MatchString(uuid) {
		return "", fmt.Errorf("couldn't resolve OriginalUUID=%v for tid=%v because it's not a valid UUID", tid, uuid)
	}

	isPresent, err := r.client.IsUUIDPresent(uuid, tid)
	if err != nil {
		return "", fmt.Errorf("couldn't resolve OriginalUUID=%v for tid=%v, error was: %v", uuid, tid, err)
	}

	if isPresent {
		return uuid, nil
	}
	return "", nil
}

func (r *httpResolver) resolveIdentifier(authority string, identifier string, tid string) (string, error) {
	status, location, err := r.client.ContentQuery(authority, identifier, tid)
	if err != nil {
		return "", err
	}
	if status != http.StatusMovedPermanently {
		return "", fmt.Errorf("unexpected response code while fetching canonical identifier for tid=%v authority=%v identifier=%v status=%v", tid, authority, identifier, status)
	}

	parts := strings.Split(location, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("resolved a canonical identifier which is an invalid FT URI for tid=%v authority=%v identifier=%v location=%v", tid, authority, identifier, location)
	}
	uuid := parts[len(parts)-1]
	if !uuidRegex.MatchString(uuid) {
		fmt.Println(parts)
		return "", fmt.Errorf("resolved a canonical identifier which contains an invalid uuid for tid=%v authority=%v identifier=%v uuid=%v", tid, authority, identifier, uuid)
	}

	return uuid, nil
}
