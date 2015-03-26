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

// APIClient that holds the required objects for API interaction
type APIClient struct {
	Key  string
	Pass string
	HTTP *http.Client
}

// StatusCode wraps HTTP status codes returned by the BitBucket API
type StatusCode int

// APIResponse holds the HTTP response from a call to the BitBucket API
type APIResponse struct {
	Header     http.Header
	StatusCode StatusCode
	Body       string
}

// Repository contains the desireds repository properties
type Repository struct {
	FullName    string `json:"full_name"`
	Description string
}

// RepositoryResponse contains the support information returned by the API
type RepositoryResponse struct {
	PageLen      int
	Size         int
	Repositories []Repository `json:"values"`
}

// DeployKey contains the desired deploy key properties
type DeployKey struct {
	ID    int `json:"pk"`
	Key   string
	Label string
}

// Service contains properties for service hooks on a repository
type Service struct {
	ID      int
	Service struct {
		Fields []struct {
			Name  string
			Value string
		}
		Type string
	}
}

type restrictionUser struct {
	Username string `json:"username"`
}

type restrictionGroup struct {
	Slug  string          `json:"slug"`
	Owner restrictionUser `json:"owner"`
}

type branchRestriction struct {
	Kind    string             `json:"kind"`
	Pattern string             `json:"pattern"`
	Groups  []restrictionGroup `json:"groups"`
	Users   []restrictionUser  `json:"users"`
}

const baseURL string = "https://bitbucket.org/api"

// New returns an API client for BitBucket
func New(key string, pass string) *APIClient {
	client := &APIClient{}

	client.Key = key
	client.Pass = pass
	client.HTTP = &http.Client{}

	return client
}

func (c *APIClient) callV1(endpoint string, method string, params url.Values) *APIResponse {
	return c.callFormEnc("1.0", endpoint, method, params)
}

func (c *APIClient) callV2(endpoint string, method string, params url.Values) *APIResponse {
	return c.callFormEnc("2.0", endpoint, method, params)
}

func (c *APIClient) callV1JSON(endpoint string, method string, params interface{}) *APIResponse {
	return c.callJSONEnc("1.0", endpoint, method, params)
}

func (c *APIClient) callV2JSON(endpoint string, method string, params interface{}) *APIResponse {
	return c.callJSONEnc("2.0", endpoint, method, params)
}

func (c *APIClient) callFormEnc(version string, endpoint string, method string, params url.Values) *APIResponse {
	if params == nil {
		params = url.Values{}
	}

	payload := params.Encode()

	return c.call(version, endpoint, method, "application/x-www-form-urlencoded", bytes.NewBufferString(payload))
}

func (c *APIClient) callJSONEnc(version string, endpoint string, method string, params interface{}) *APIResponse {
	payload, _ := json.Marshal(params)

	return c.call(version, endpoint, method, "application/json", bytes.NewBuffer(payload))
}

func (c *APIClient) call(version string, endpoint string, method string, contentType string, payload *bytes.Buffer) *APIResponse {
	apiurl := fmt.Sprintf("%s/%s/%s", baseURL, version, endpoint)

	req, _ := http.NewRequest(method, apiurl, payload)

	if method != "GET" {
		req.Header.Add("Content-Type", contentType+"; charset=utf-8")
		req.Header.Add("Content-Length", strconv.Itoa(payload.Len()))
	}

	req.SetBasicAuth(c.Key, c.Pass)
	resp, _ := c.HTTP.Do(req)
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	apiresp := &APIResponse{resp.Header, StatusCode(resp.StatusCode), string(body)}

	return apiresp
}

// GetRepositories returns a list of all repositories owned by `owner`
func (c *APIClient) GetRepositories(owner string) ([]Repository, error) {
	var repos []Repository

	page := 1
	for {
		apiresp := c.callV2(fmt.Sprintf("repositories/%s?page=%d", owner, page), "GET", nil)

		var reporesp RepositoryResponse
		json.Unmarshal([]byte(apiresp.Body), &reporesp)

		repos = append(repos, reporesp.Repositories...)

		if page*reporesp.PageLen > reporesp.Size {
			break
		}

		page++
	}

	return repos, nil
}

// RepositoriesChanged returns whether or not the repositories for an account has changed
// as well as the latest ETag returned by the web server.
func (c *APIClient) RepositoriesChanged(owner string, etag string) (bool, string, error) {
	apiresp := c.callV2(fmt.Sprintf("repositories/%s", owner), "HEAD", nil)

	if apiresp.StatusCode != 200 {
		return false, etag, fmt.Errorf("%s", apiresp.Body)
	}

	currentEtag := apiresp.Header.Get("Etag")

	return etag != currentEtag, currentEtag, nil
}

// PostBranchRestriction adds a new branch restriction to a repository
func (c *APIClient) PostBranchRestriction(owner string, repo string, kind string, branchpattern string, users []string, groups []string) error {
	restriction := branchRestriction{}
	restriction.Kind = kind
	restriction.Pattern = branchpattern

	if users != nil {
		for _, username := range users {
			restriction.Users = append(restriction.Users, restrictionUser{username})
		}
	}

	if groups != nil {
		for _, groupname := range groups {
			restriction.Groups = append(restriction.Groups, restrictionGroup{groupname, restrictionUser{owner}})
		}
	}

	apiresp := c.callV2JSON(fmt.Sprintf("repositories/%s/%s/branch-restrictions", owner, repo), "POST", restriction)

	if apiresp.StatusCode == 200 || apiresp.StatusCode == 409 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", apiresp.StatusCode, apiresp.Body)
}

// GetDeployKeys returns a list of all deploy keys attached to a repository
func (c *APIClient) GetDeployKeys(owner string, repo string) ([]DeployKey, error) {
	apiresp := c.callV1(fmt.Sprintf("repositories/%s/%s/deploy-keys", owner, repo), "GET", nil)

	if apiresp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", apiresp.Body)
	}

	var keys []DeployKey

	json.Unmarshal([]byte(apiresp.Body), &keys)

	return keys, nil
}

// GetServices returns a list of the service hooks attached to a repository
func (c *APIClient) GetServices(owner string, repository string) ([]Service, error) {
	resp := c.callV1(fmt.Sprintf("repositories/%s/%s/services", owner, repository), "GET", nil)

	if resp.StatusCode != 200 {

	}

	var serviceResponse []Service

	json.Unmarshal([]byte(resp.Body), &serviceResponse)

	return serviceResponse, nil
}

// PostService attaches a new service hook to the repository
func (c *APIClient) PostService(owner string, repository string, servicetype string, parameters map[string]string) error {
	data := url.Values{}
	data.Set("type", servicetype)

	for key, value := range parameters {
		data.Set(key, value)
	}

	resp := c.callV1(fmt.Sprintf("repositories/%s/%s/services", owner, repository), "POST", data)

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// PostDeployKey attaches a new deploy key to a repository
func (c *APIClient) PostDeployKey(owner string, repository string, name string, key string) error {
	data := url.Values{}
	data.Set("label", name)
	data.Set("key", key)

	resp := c.callV1(fmt.Sprintf("repositories/%s/%s/deploy-keys", owner, repository), "POST", data)

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// DeleteDeployKey removes a deploy key from a repository
func (c *APIClient) DeleteDeployKey(owner string, repository string, keyID int) error {
	resp := c.callV1(fmt.Sprintf("repositories/%s/%s/deploy-keys/%d", owner, repository, keyID), "DELETE", nil)

	if resp.StatusCode == 204 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// Used when updating properties on repositories
func (c *APIClient) putV1RepoProp(owner string, repository string, data url.Values) *APIResponse {
	return c.callV1(fmt.Sprintf("repositories/%s/%s", owner, repository), "PUT", data)
}

func (c *APIClient) getV1Error(resp *APIResponse) error {
	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// PutLandingPage sets the landing page for a repository: "branches", "commits",
// "downloads", "overview", "pull_requests" or "source".
func (c *APIClient) PutLandingPage(owner string, repository string, landingpage string) error {
	data := url.Values{}
	data.Set("landing_page", landingpage)

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

// PutPrivacy set the repository privacy/visibility
func (c *APIClient) PutPrivacy(owner string, repository string, isPrivate bool) error {
	data := url.Values{}
	data.Set("is_private", fmt.Sprintf("%t", isPrivate))

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

// PutPublicIssueTracker sets whether the repository has PUBLIC or NO issue tracker
// (Private issue trackers doesn't seem to be supported by the API)
func (c *APIClient) PutPublicIssueTracker(owner string, repository string, hasPublicIssueTracker bool) error {
	data := url.Values{}
	data.Set("has_issues", fmt.Sprintf("%t", hasPublicIssueTracker))

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

// PutMainBranch sets the main branch for the repository
func (c *APIClient) PutMainBranch(owner string, repository string, mainBranch string) error {
	data := url.Values{}
	data.Set("main_branch", mainBranch)

	res := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res)
}

// PutForks set the forking policy for the repository: "none", "private" or "public"
func (c *APIClient) PutForks(owner string, repository string, forks string) error {
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
