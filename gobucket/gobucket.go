package gobucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

func New(key string, pass string) *ApiClient {
	client := &ApiClient{}

	client.Key = key
	client.Pass = pass
	client.Http = &http.Client{}

	return client
}

type ApiClient struct {
	Key  string
	Pass string
	Http *http.Client
}

type StatusCode int

type ApiResponse struct {
	Header     http.Header
	StatusCode StatusCode
	Body       string
}

type Repository struct {
	FullName    string `json:"full_name"`
	Description string
}

type RepositoryResponse struct {
	PageLen      int
	Size         int
	Repositories []Repository `json:"values"`
}

const baseUrl string = "https://bitbucket.org/api"

func (c *ApiClient) callV1(endpoint string, method string, params url.Values) *ApiResponse {
	return c.call("1.0", endpoint, method, params)
}

func (c *ApiClient) callV2(endpoint string, method string, params url.Values) *ApiResponse {
	return c.call("2.0", endpoint, method, params)
}

func (c *ApiClient) call(version string, endpoint string, method string, params url.Values) *ApiResponse {
	apiurl := fmt.Sprintf("%s/%s/%s", baseUrl, version, endpoint)

	if params == nil {
		params = url.Values{}
	}

	req, _ := http.NewRequest(method, apiurl, bytes.NewBufferString(params.Encode()))

	if method != "GET" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(params.Encode())))
	}

	req.SetBasicAuth(c.Key, c.Pass)
	resp, _ := c.Http.Do(req)
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	apiresp := &ApiResponse{resp.Header, StatusCode(resp.StatusCode), string(body)}

	return apiresp
}

func (c *ApiClient) GetRepositories(owner string) []Repository {
	var repos []Repository = make([]Repository, 0)

	page := 1
	for {
		apiresp := c.callV2(fmt.Sprintf("repositories/%s?page=%d", owner, page), "GET", nil)

		var reporesp RepositoryResponse
		json.Unmarshal([]byte(apiresp.Body), &reporesp)

		repos = append(repos, reporesp.Repositories...)

		if page*reporesp.PageLen > reporesp.Size {
			break
		}

		page += 1
	}

	return repos
}

func (c *ApiClient) RepositoriesChanged(owner string, etag string) (bool, string) {
	apiresp := c.callV2(fmt.Sprintf("repositories/%s", owner), "HEAD", nil)

	curr_etag := apiresp.Header.Get("Etag")

	return etag != curr_etag, curr_etag
}

func (c *ApiClient) PutLandingPage(owner string, repository string, landingpage string) bool {
	data := url.Values{}
	data.Set("landing_page", landingpage)
	return c.putV1RepoProp(owner, repository, data).StatusCode == 200
}

func (c *ApiClient) PutPrivacy(owner string, repository string, is_private bool, private_forks bool) bool {
	data := url.Values{}
	data.Set("is_private", fmt.Sprintf("%t", is_private))

	if is_private {
		data.Set("no_forks", fmt.Sprintf("%t", private_forks))
		data.Set("no_public_forks", "true")
	} else {
		data.Set("no_forks", "false")
		data.Set("no_public_forks", "false")
	}

	return c.putV1RepoProp(owner, repository, data).StatusCode == 200
}

func (c *ApiClient) putV1RepoProp(owner string, repository string, data url.Values) *ApiResponse {
	return c.callV1(fmt.Sprintf("repositories/%s/%s", owner, repository), "PUT", data)
}
