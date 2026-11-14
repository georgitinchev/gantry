// Package clients is a minimal HTTP client for the gantry-cloud API.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIError is a non-2xx response from gantry-cloud.
type APIError struct {
	Status   int
	Code     string          `json:"error"`
	Message  string          `json:"message"`
	Existing json.RawMessage `json:"existing,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gantry-cloud %d %s: %s", e.Status, e.Code, e.Message)
}

func IsNotFound(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

// IsConflict reports a duplicate-name create; Existing holds the object.
func IsConflict(err error) (*APIError, bool) {
	var e *APIError
	ok := errors.As(err, &e) && e.Code == "conflict"
	return e, ok
}

type Network struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}

type Instance struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NetworkID string `json:"networkId"`
	Size      string `json:"size"`
	IP        string `json:"ip"`
	State     string `json:"state"`
}

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{BaseURL: baseURL, Token: token, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		e := &APIError{Status: res.StatusCode}
		_ = json.Unmarshal(data, e)
		return e
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *Client) GetNetwork(ctx context.Context, id string) (*Network, error) {
	n := &Network{}
	return n, c.do(ctx, http.MethodGet, "/v1/networks/"+id, nil, n)
}

func (c *Client) CreateNetwork(ctx context.Context, name, cidr string) (*Network, error) {
	n := &Network{}
	return n, c.do(ctx, http.MethodPost, "/v1/networks", map[string]string{"name": name, "cidr": cidr}, n)
}

func (c *Client) DeleteNetwork(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/networks/"+id, nil, nil)
}

func (c *Client) GetInstance(ctx context.Context, id string) (*Instance, error) {
	i := &Instance{}
	return i, c.do(ctx, http.MethodGet, "/v1/instances/"+id, nil, i)
}

func (c *Client) CreateInstance(ctx context.Context, name, networkID, size string) (*Instance, error) {
	i := &Instance{}
	return i, c.do(ctx, http.MethodPost, "/v1/instances", map[string]string{"name": name, "networkId": networkID, "size": size}, i)
}

func (c *Client) ResizeInstance(ctx context.Context, id, size string) (*Instance, error) {
	i := &Instance{}
	return i, c.do(ctx, http.MethodPatch, "/v1/instances/"+id, map[string]string{"size": size}, i)
}

func (c *Client) DeleteInstance(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/instances/"+id, nil, nil)
}
