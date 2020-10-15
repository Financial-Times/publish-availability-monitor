package checks

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

type DocStoreClient interface {
	ContentQuery(authority string, identifier string, tid string) (status int, location string, err error)
	IsUUIDPresent(uuid, tid string) (isPresent bool, err error)
}

type httpDocStoreClient struct {
	docStoreAddress string
	httpCaller      HttpCaller
	username        string
	password        string
}

func NewHttpDocStoreClient(docStoreAddress string, httpCaller HttpCaller, username, password string) *httpDocStoreClient {
	return &httpDocStoreClient{
		docStoreAddress: docStoreAddress,
		httpCaller:      httpCaller,
		username:        username,
		password:        password,
	}
}

func (c *httpDocStoreClient) ContentQuery(authority string, identifier string, tid string) (status int, location string, err error) {
	docStoreUrl, err := url.Parse(c.docStoreAddress + "/content-query")
	if err != nil {
		return -1, "", fmt.Errorf("invalid address docStoreAddress=%v", c.docStoreAddress)
	}
	query := url.Values{}
	query.Add("identifierValue", identifier)
	query.Add("identifierAuthority", authority)
	docStoreUrl.RawQuery = query.Encode()

	resp, err := c.httpCaller.DoCall(Config{
		Url:      docStoreUrl.String(),
		Username: c.username,
		Password: c.password,
		TxId:     ConstructPamTxId(tid),
	})

	if err != nil {
		return -1, "", fmt.Errorf("unsuccessful request for fetching canonical identifier for authority=%v identifier=%v url=%v, error was: %v", authority, identifier, docStoreUrl.String(), err.Error())
	}
	niceClose(resp)

	return resp.StatusCode, resp.Header.Get("Location"), nil
}

func (c *httpDocStoreClient) IsUUIDPresent(uuid, tid string) (isPresent bool, err error) {
	docStoreUrl, err := url.Parse(c.docStoreAddress + "/content/" + uuid)
	if err != nil {
		return false, fmt.Errorf("invalid address docStoreAddress=%v", c.docStoreAddress)
	}

	resp, err := c.httpCaller.DoCall(Config{
		Url:      docStoreUrl.String(),
		Username: c.username,
		Password: c.password,
		TxId:     ConstructPamTxId(tid),
	})

	if err != nil {
		return false, fmt.Errorf("failed to check the presence of UUID=%v in document-store, error was: %v", uuid, err.Error())
	}
	niceClose(resp)

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("failed to check presence of UUID=%v in document-store, service request returned StatusCode=%v", uuid, resp.StatusCode)
}

func niceClose(resp *http.Response) {
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("Couldn't close response body %v", err)
		}
	}()
}
