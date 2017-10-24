package checks

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
)

type DocStoreClient interface {
	ContentQuery(authority string, identifier string, tid string) (status int, location string, err error)
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
		return -1, "", fmt.Errorf("Invalid address docStoreAddress=%v", c.docStoreAddress)
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
		return -1, "", fmt.Errorf("Unsuccessful request for fetching canonical identifier for authority=%v identifier=%v url=%v, error was: %v", authority, identifier, docStoreUrl.String(), err.Error())
	}
	niceClose(resp)

	return resp.StatusCode, resp.Header.Get("Location"), nil
}

func niceClose(resp *http.Response) {
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Warnf("Couldn't close response body %v", err)
		}
	}()
}
