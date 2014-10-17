package gobucket

import (
//  "net/http"
  "fmt"
)

func New(key string, pass string) *ApiClient {
  client := &ApiClient{}

  client.key = key
  client.pass = pass

  return client
}

type ApiClient struct {
  key string
  pass string
}

type Method int

const (
  GET Method = iota
  POST
  PUT
  DELETE
)

const baseUrl string = "https://bitbucket.org/api"

func (c *ApiClient) callV1(endpoint string, params map[string]string) string {
  return c.call("1.0", endpoint, params)
}

func (c *ApiClient) callV2(endpoint string, params map[string]string) string {
  return c.call("2.0", endpoint, params)
}

func (c *ApiClient) call(version string, endpoint string, params map[string]string) string {
  url := fmt.Sprintf("%s/%s/%s", baseUrl, version, endpoint)

  return url
}

func (c *ApiClient) GetRepositories(owner string) string {
  return c.callV2(fmt.Sprintf("repositories/%s", owner), nil)
}
