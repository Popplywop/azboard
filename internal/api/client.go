package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/popplywop/azboard/internal/config"
)

type Client struct {
	baseURL    string
	orgURL     string // org-level URL for endpoints like connectionData
	project    string // Azure DevOps project name
	httpClient *http.Client
	authMethod config.AuthMethod
	token      string
	tokenExp   time.Time
	mu         sync.Mutex
}

type azTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresOn   string `json:"expiresOn"`
}

const adoResource = "499b84ac-1321-427f-aa17-267ca6975798"

// NewClient creates an API client from the loaded config.
func NewClient(cfg *config.Config) (*Client, error) {
	// Build org-level URL — handle both dev.azure.com and visualstudio.com
	orgURL := fmt.Sprintf("https://dev.azure.com/%s/_apis", cfg.Org)
	if cfg.OrgURL != "" {
		// Use the configured org URL directly (strip trailing slash, add _apis)
		base := strings.TrimRight(cfg.OrgURL, "/")
		// Remove project from URL if present (e.g. https://pdidev.visualstudio.com/PDI -> https://pdidev.visualstudio.com)
		if strings.HasSuffix(strings.ToLower(base), "/"+strings.ToLower(cfg.Project)) {
			base = base[:len(base)-len(cfg.Project)-1]
		}
		orgURL = base + "/_apis"
	}

	// Build project-scoped baseURL from orgURL when set, otherwise use dev.azure.com default.
	baseURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis", cfg.Org, cfg.Project)
	if cfg.OrgURL != "" {
		base := strings.TrimRight(cfg.OrgURL, "/")
		// Strip project suffix if it was included in OrgURL
		if strings.HasSuffix(strings.ToLower(base), "/"+strings.ToLower(cfg.Project)) {
			base = base[:len(base)-len(cfg.Project)-1]
		}
		baseURL = base + "/" + cfg.Project + "/_apis"
	}

	c := &Client{
		baseURL:    baseURL,
		orgURL:     orgURL,
		project:    cfg.Project,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		authMethod: cfg.AuthMethod,
	}

	switch cfg.AuthMethod {
	case config.AuthPAT:
		c.token = cfg.PAT
		// PATs don't expire (well, they do eventually, but not on a token-refresh cycle)
		c.tokenExp = time.Now().Add(365 * 24 * time.Hour)
	case config.AuthAzCLI:
		if err := c.refreshAzCLIToken(); err != nil {
			return nil, fmt.Errorf("failed to get Azure DevOps token: %w", err)
		}
	}

	return c, nil
}

func (c *Client) refreshAzCLIToken() error {
	cmd := exec.Command("az", "account", "get-access-token",
		"--resource", adoResource,
		"--output", "json",
	)

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("az account get-access-token failed: %w (is az cli logged in?)", err)
	}

	var resp azTokenResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	c.token = resp.AccessToken

	// Parse expiry — az cli returns "2024-01-15 12:00:00.000000" format
	exp, err := time.Parse("2006-01-02 15:04:05.000000", resp.ExpiresOn)
	if err != nil {
		// If we can't parse, set expiry to 50 minutes from now (tokens last ~60 min)
		c.tokenExp = time.Now().Add(50 * time.Minute)
	} else {
		c.tokenExp = exp
	}

	return nil
}

func (c *Client) ensureToken() error {
	// PATs don't need refreshing
	if c.authMethod == config.AuthPAT {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Refresh if token expires within 5 minutes
	if time.Until(c.tokenExp) < 5*time.Minute {
		return c.refreshAzCLIToken()
	}
	return nil
}

func (c *Client) authHeader() string {
	if c.authMethod == config.AuthPAT {
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+c.token))
	}
	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(":" + tok))
}

// doRequest executes an HTTP request with auth, api-version, and 401 retry.
func (c *Client) doRequest(method, fullURL string, body interface{}, result interface{}) error {
	return c.doRequestWithVersion(method, fullURL, "7.1", body, result)
}

// doRequestWithVersion is like doRequest but lets the caller specify the api-version string
// (e.g. "7.1-preview.4" for preview endpoints). An optional contentType overrides
// the default "application/json" (used by patchJSONPatch for "application/json-patch+json").
func (c *Client) doRequestWithVersion(method, fullURL, apiVersion string, body interface{}, result interface{}, contentType ...string) error {
	if err := c.ensureToken(); err != nil {
		return err
	}

	ct := "application/json"
	if len(contentType) > 0 && contentType[0] != "" {
		ct = contentType[0]
	}

	// Append api-version
	if strings.Contains(fullURL, "?") {
		fullURL += "&api-version=" + apiVersion
	} else {
		fullURL += "?api-version=" + apiVersion
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", ct)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Close the 401 response body explicitly before retrying to avoid
		// leaking the connection (a deferred close on the reassigned resp
		// variable would miss this body).
		resp.Body.Close()
		if c.authMethod == config.AuthAzCLI {
			c.mu.Lock()
			err := c.refreshAzCLIToken()
			c.mu.Unlock()
			if err != nil {
				return fmt.Errorf("token refresh failed: %w", err)
			}
			// Rebuild request for retry (body may have been consumed)
			if body != nil {
				jsonBytes, err := json.Marshal(body)
				if err != nil {
					return fmt.Errorf("failed to marshal request body on retry: %w", err)
				}
				bodyReader = bytes.NewReader(jsonBytes)
			}
			req, err = http.NewRequest(method, fullURL, bodyReader)
			if err != nil {
				return fmt.Errorf("failed to create retry request: %w", err)
			}
			req.Header.Set("Authorization", c.authHeader())
			req.Header.Set("Accept", "application/json")
			if body != nil {
				req.Header.Set("Content-Type", ct)
			}
			resp, err = c.httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("retry request failed: %w", err)
			}
			defer resp.Body.Close()
		} else {
			return fmt.Errorf("authentication failed — check that your PAT is valid and has not expired")
		}
	} else {
		defer resp.Body.Close()
	}

	// Accept 2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) get(path string, result interface{}) error {
	return c.doRequest("GET", c.baseURL+path, nil, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	return c.doRequest("POST", c.baseURL+path, body, result)
}

func (c *Client) getPreview(path, version string, result interface{}) error {
	return c.doRequestWithVersion("GET", c.baseURL+path, version, nil, result)
}

func (c *Client) postPreview(path, version string, body interface{}, result interface{}) error {
	return c.doRequestWithVersion("POST", c.baseURL+path, version, body, result)
}

func (c *Client) put(path string, body interface{}, result interface{}) error {
	return c.doRequest("PUT", c.baseURL+path, body, result)
}

func (c *Client) patch(path string, body interface{}, result interface{}) error {
	return c.doRequest("PATCH", c.baseURL+path, body, result)
}

// getOrg makes a GET request against the org-level URL (no project in path).
func (c *Client) getOrg(path string, result interface{}) error {
	return c.doRequest("GET", c.orgURL+path, nil, result)
}

// patchJSONPatch makes a PATCH request with Content-Type application/json-patch+json.
// Used for work item update operations.
func (c *Client) patchJSONPatch(path string, body interface{}, result interface{}) error {
	return c.doRequestWithVersion("PATCH", c.baseURL+path, "7.1", body, result, "application/json-patch+json")
}

// getContent fetches raw file content (not JSON-decoded) for a given path.
// Used for git item endpoints where $format=text returns plain bytes.
func (c *Client) getContent(path string) (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}

	fullURL := c.baseURL + path
	if strings.Contains(fullURL, "?") {
		fullURL += "&api-version=7.1"
	} else {
		fullURL += "?api-version=7.1"
	}

	doGet := func() (*http.Response, error) {
		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", c.authHeader())
		req.Header.Set("Accept", "text/plain")
		return c.httpClient.Do(req)
	}

	resp, err := doGet()
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if c.authMethod == config.AuthAzCLI {
			c.mu.Lock()
			err = c.refreshAzCLIToken()
			c.mu.Unlock()
			if err != nil {
				return "", fmt.Errorf("token refresh failed: %w", err)
			}
			resp, err = doGet()
			if err != nil {
				return "", fmt.Errorf("retry request failed: %w", err)
			}
			defer resp.Body.Close()
		} else {
			return "", fmt.Errorf("authentication failed — check that your PAT is valid and has not expired")
		}
	} else {
		defer resp.Body.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(data), nil
}
