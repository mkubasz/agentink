// Package client is how `ink` talks to `inkd`: JSON over HTTP on the daemon's
// Unix socket. It knows nothing of the worker (ADR 0007).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/mkubasz/agentink/internal/platform/api"
)

// ErrUnreachable means no daemon answered on the socket.
var ErrUnreachable = errors.New("inkd is not running")

// APIError is a non-2xx answer from the daemon.
type APIError struct {
	Status int
	Msg    string
}

func (e *APIError) Error() string { return fmt.Sprintf("inkd: %d: %s", e.Status, e.Msg) }

// Client calls the daemon.
type Client struct {
	http *http.Client
	base string
}

// New returns a client for the daemon on socketPath.
func New(socketPath string) *Client {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer

			return d.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{http: &http.Client{Transport: tr, Timeout: 30 * time.Second}, base: "http://inkd"}
}

// NewHTTP returns a client for a daemon reachable over plain HTTP (tests).
func NewHTTP(base string, hc *http.Client) *Client {
	return &Client{http: hc, base: base}
}

// Status asks the daemon how it is.
func (c *Client) Status(ctx context.Context) (api.Status, error) {
	var out api.Status

	return out, c.call(ctx, http.MethodGet, "/v1/status", nil, &out)
}

// Ask submits a question.
func (c *Client) Ask(ctx context.Context, req api.AskRequest) (api.AskResponse, error) {
	var out api.AskResponse

	return out, c.call(ctx, http.MethodPost, "/v1/questions", req, &out)
}

// Inbox lists the open questions, for one project when project is not empty.
func (c *Client) Inbox(ctx context.Context, project string) ([]api.InboxRow, error) {
	path := "/v1/inbox"
	if project != "" {
		path += "?project=" + url.QueryEscape(project)
	}

	var out []api.InboxRow

	return out, c.call(ctx, http.MethodGet, path, nil, &out)
}

// Question returns where one question stands, with its answer once confirmed.
func (c *Client) Question(ctx context.Context, id string) (api.Question, error) {
	var out api.Question

	return out, c.call(ctx, http.MethodGet, "/v1/questions/"+url.PathEscape(id), nil, &out)
}

// AddDevice registers a reader and returns its token once.
func (c *Client) AddDevice(ctx context.Context, req api.AddDeviceRequest) (api.AddDeviceResponse, error) {
	var out api.AddDeviceResponse

	return out, c.call(ctx, http.MethodPost, "/v1/devices", req, &out)
}

// RevokeDevice ends a reader's access.
func (c *Client) RevokeDevice(ctx context.Context, name string) error {
	var out struct{}

	return c.call(ctx, http.MethodDelete, "/v1/devices/"+url.PathEscape(name), nil, &out)
}

// Devices lists the readers.
func (c *Client) Devices(ctx context.Context) ([]api.Device, error) {
	var out []api.Device

	return out, c.call(ctx, http.MethodGet, "/v1/devices", nil, &out)
}

// RegisterProject adds a project.
func (c *Client) RegisterProject(ctx context.Context, req api.RegisterProjectRequest) (api.Project, error) {
	var out api.Project

	return out, c.call(ctx, http.MethodPost, "/v1/projects", req, &out)
}

// Projects lists the registered projects.
func (c *Client) Projects(ctx context.Context) ([]api.Project, error) {
	var out []api.Project

	return out, c.call(ctx, http.MethodGet, "/v1/projects", nil, &out)
}

func (c *Client) call(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader

	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("client: marshal: %w", err)
		}

		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return fmt.Errorf("client: request: %w", err)
	}

	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	}

	defer resp.Body.Close() //nolint:errcheck // read-only response body

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("client: read: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		var apiErr api.Error

		_ = json.Unmarshal(raw, &apiErr)

		if apiErr.Error == "" {
			apiErr.Error = http.StatusText(resp.StatusCode)
		}

		return &APIError{Status: resp.StatusCode, Msg: apiErr.Error}
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		return nil // 204 No Content
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("client: decode: %w", err)
	}

	return nil
}
