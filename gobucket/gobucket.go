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

func (c *APIClient) callFormEnc(version string, endpoint string, method string, params url.Values) (*APIResponse, error) {
	if params == nil {
		params = url.Values{}
	}

	payload := params.Encode()

	return c.call(version, endpoint, method, "application/x-www-form-urlencoded", bytes.NewBufferString(payload))
}

func (c *APIClient) callJSONEnc(version string, endpoint string, method string, params interface{}) (*APIResponse, error) {
	payload, _ := json.Marshal(params)

	return c.call(version, endpoint, method, "application/json", bytes.NewBuffer(payload))
}

func (c *APIClient) callStringBody(version string, endpoint string, method string, payload string) (*APIResponse, error) {
	return c.call(version, endpoint, method, "text/plain", bytes.NewBufferString(payload))
}

func (c *APIClient) call(version string, endpoint string, method string, contentType string, payload *bytes.Buffer) (*APIResponse, error) {
	apiurl := fmt.Sprintf("%s/%s/%s", baseURL, version, endpoint)

	req, err := http.NewRequest(method, apiurl, payload)

	if err != nil {
		return nil, err
	}

	req.Close = true

	if method != "GET" {
		req.Header.Add("Content-Type", contentType+"; charset=utf-8")
		req.Header.Add("Content-Length", strconv.Itoa(payload.Len()))
	}

	req.SetBasicAuth(c.Key, c.Pass)
	resp, err := c.HTTP.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	apiresp := &APIResponse{resp.Header, StatusCode(resp.StatusCode), string(body)}

	return apiresp, nil
}

// GetRepositories returns a list of all repositories owned by `owner`
func (c *APIClient) GetRepositories(owner string) ([]Repository, error) {
	var repos []Repository

	page := 1
	for {
		apiresp, err := c.call("2.0", fmt.Sprintf("repositories/%s?page=%d", owner, page), "GET", "", &bytes.Buffer{})
		if err != nil {
			return []Repository{}, err
		}

		var reporesp RepositoryResponse
		json.Unmarshal([]byte(apiresp.Body), &reporesp)

		repos = append(repos, reporesp.Repositories...)

		if page*reporesp.PageLen >= reporesp.Size {
			break
		}

		page++
	}

	return repos, nil
}

// RepositoriesChanged returns whether or not the repositories for an account has changed
// as well as the latest ETag returned by the web server.
func (c *APIClient) RepositoriesChanged(owner string, etag string) (bool, string, error) {
	apiresp, err := c.callFormEnc("2.0", fmt.Sprintf("repositories/%s", owner), "HEAD", nil)

	if err != nil {
		return false, etag, err
	}

	if apiresp.StatusCode != 200 {
		return false, etag, fmt.Errorf("%s", apiresp.Body)
	}

	currentEtag := apiresp.Header.Get("Etag")

	return etag != currentEtag, currentEtag, nil
}

// AddBranchRestriction adds a new branch restriction to a repository
func (c *APIClient) AddBranchRestriction(owner string, repo string, kind string, branchpattern string, users []string, groups []string) error {
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

	apiresp, err := c.callJSONEnc("2.0", fmt.Sprintf("repositories/%s/%s/branch-restrictions", owner, repo), "POST", restriction)

	if err != nil {
		return err
	}

	if apiresp.StatusCode == 200 || apiresp.StatusCode == 409 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", apiresp.StatusCode, apiresp.Body)
}

// AddUserPrivilege adds a privilege for a user on a repository
func (c *APIClient) AddUserPrivilege(owner string, repo string, privilegeUser string, privilege string) error {
	return c.addPrivilege(owner, repo, "privileges", privilegeUser, privilege)
}

// AddGroupPrivilege adds a privilege for a group on a repository
func (c *APIClient) AddGroupPrivilege(owner string, repo string, privilegeGroup string, privilege string) error {
	group := fmt.Sprintf("%s/%s", owner, privilegeGroup)
	return c.addPrivilege(owner, repo, "group-privileges", group, privilege)
}

func (c *APIClient) addPrivilege(owner string, repo string, privilegeURL string, privilegeEntity string, privilege string) error {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", privilegeURL, owner, repo, privilegeEntity)

	if !(privilege == "read" || privilege == "write" || privilege == "admin") {
		return fmt.Errorf("Wrong privilege ('%s'). One of 'read', 'write' or 'admin' required.", privilege)
	}

	apiresp, err := c.callStringBody("1.0", endpoint, "PUT", privilege)

	if err != nil {
		return err
	}

	if apiresp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", apiresp.StatusCode, apiresp.Body)
}

// GetServices returns a list of the service hooks attached to a repository
func (c *APIClient) GetServices(owner string, repository string) ([]Service, error) {
	resp, err := c.callFormEnc("1.0", fmt.Sprintf("repositories/%s/%s/services", owner, repository), "GET", nil)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
	}

	var serviceResponse []Service

	json.Unmarshal([]byte(resp.Body), &serviceResponse)

	return serviceResponse, nil
}

// AddService attaches a new service hook to the repository
func (c *APIClient) AddService(owner string, repository string, servicetype string, parameters map[string]string) error {
	data := url.Values{}
	data.Set("type", servicetype)

	for key, value := range parameters {
		data.Set(key, value)
	}

	resp, err := c.callFormEnc("1.0", fmt.Sprintf("repositories/%s/%s/services", owner, repository), "POST", data)

	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// GetDeployKeys returns a list of all deploy keys attached to a repository
func (c *APIClient) GetDeployKeys(owner string, repo string) ([]DeployKey, error) {
	apiresp, err := c.callFormEnc("1.0", fmt.Sprintf("repositories/%s/%s/deploy-keys", owner, repo), "GET", nil)

	if err != nil {
		return nil, err
	}

	if apiresp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", apiresp.Body)
	}

	var keys []DeployKey

	json.Unmarshal([]byte(apiresp.Body), &keys)

	return keys, nil
}

// AddDeployKey attaches a new deploy key to a repository
func (c *APIClient) AddDeployKey(owner string, repository string, name string, key string) error {
	data := url.Values{}
	data.Set("label", name)
	data.Set("key", key)

	resp, err := c.callFormEnc("1.0", fmt.Sprintf("repositories/%s/%s/deploy-keys", owner, repository), "POST", data)

	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// DeleteDeployKey removes a deploy key from a repository
func (c *APIClient) DeleteDeployKey(owner string, repository string, keyID int) error {
	resp, err := c.callFormEnc("1.0", fmt.Sprintf("repositories/%s/%s/deploy-keys/%d", owner, repository, keyID), "DELETE", nil)

	if err != nil {
		return err
	}

	if resp.StatusCode == 204 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// Used when updating properties on repositories
func (c *APIClient) putV1RepoProp(owner string, repository string, data url.Values) (*APIResponse, error) {
	return c.callFormEnc("1.0", fmt.Sprintf("repositories/%s/%s", owner, repository), "PUT", data)
}

func (c *APIClient) getV1Error(resp *APIResponse, err error) error {
	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("[%d]: %s", resp.StatusCode, resp.Body)
}

// SetLandingPage sets the landing page for a repository: "branches", "commits",
// "downloads", "overview", "pull_requests" or "source".
func (c *APIClient) SetLandingPage(owner string, repository string, landingpage string) error {
	data := url.Values{}
	data.Set("landing_page", landingpage)

	res, err := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res, err)
}

// SetPrivacy set the repository privacy/visibility
func (c *APIClient) SetPrivacy(owner string, repository string, isPrivate bool) error {
	data := url.Values{}
	data.Set("is_private", fmt.Sprintf("%t", isPrivate))

	res, err := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res, err)
}

// SetIssueTracker sets whether the repository has PUBLIC or NO issue tracker
// (Private issue trackers doesn't seem to be supported by the API)
func (c *APIClient) SetIssueTracker(owner string, repository string, issueTracker string) error {
	data := url.Values{}

	if issueTracker == "none" {
		data.Set("has_issues", "false")
	} else if issueTracker == "public" {
		data.Set("has_issues", "true")
	} else {
		return fmt.Errorf("Issue tracker setting '%s' not valid. 'none' or 'public' required", issueTracker)
	}

	res, err := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res, err)
}

// SetMainBranch sets the main branch for the repository
func (c *APIClient) SetMainBranch(owner string, repository string, mainBranch string) error {
	data := url.Values{}
	data.Set("main_branch", mainBranch)

	res, err := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res, err)
}

// SetDescription sets the main branch for the repository
func (c *APIClient) SetDescription(owner string, repository string, description string) error {
	data := url.Values{}
	data.Set("description", description)

	res, err := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res, err)
}

// SetForks set the forking policy for the repository: "none", "private" or "public"
func (c *APIClient) SetForks(owner string, repository string, forks string) error {
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

	res, err := c.putV1RepoProp(owner, repository, data)
	return c.getV1Error(res, err)
}
