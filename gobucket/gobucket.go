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

type DeployKey struct {
	Id    int `json:"pk"`
	Key   string
	Label string
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

func (c *ApiClient) GetRepositories(owner string) ([]Repository, error) {
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

	return repos, nil
}

func (c *ApiClient) GetDeployKeys(owner string, repo string) ([]DeployKey, error) {
	apiresp := c.callV1(fmt.Sprintf("repositories/%s/%sasd/deploy-keys", owner, repo), "GET", nil)

	if apiresp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", apiresp.Body)
	}

	var keys []DeployKey

	json.Unmarshal([]byte(apiresp.Body), &keys)

	return keys, nil
}

func (c *ApiClient) RepositoriesChanged(owner string, etag string) (bool, string, error) {
	apiresp := c.callV2(fmt.Sprintf("repositories/%s", owner), "HEAD", nil)

	if apiresp.StatusCode != 200 {
		return false, etag, fmt.Errorf("%s", apiresp.Body)
	}

	curr_etag := apiresp.Header.Get("Etag")

	return etag != curr_etag, curr_etag, nil
}

func (c *ApiClient) PutLandingPage(owner string, repository string, landingpage string) error {
	data := url.Values{}
	data.Set("landing_page", landingpage)

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

func (c *ApiClient) PutPrivacy(owner string, repository string, is_private bool) error {
	data := url.Values{}
	data.Set("is_private", fmt.Sprintf("%t", is_private))

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

func (c *ApiClient) PutMainBranch(owner string, repository string, main_branch string) error {
	data := url.Values{}
	data.Set("main_branch", "foobar"+main_branch)

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

func (c *ApiClient) PutForks(owner string, repository string, forks string) error {
	data := url.Values{}

	if forks == "none" {
		data.Set("no_forks", "True")
		data.Set("no_public_forks", "True")
	} else if forks == "private" {
		data.Set("no_forks", "False")
		data.Set("no_public_forks", "True")
	} else if forks == "public" {
		data.Set("no_forks", "False")
		data.Set("no_public_forks", "False")
	}

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

func (c *ApiClient) putV1RepoProp(owner string, repository string, data url.Values) *ApiResponse {
	return c.callV1(fmt.Sprintf("repositories/%s/%s", owner, repository), "PUT", data)
}

func (c *ApiClient) getV1Error(resp *ApiResponse) error {
	if resp.StatusCode == 200 {
		return nil
	} else {
		return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
	}
}
