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

	"azboard/internal/config"
)

type Client struct {
	baseURL    string
	orgURL     string // org-level URL for endpoints like connectionData
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

	c := &Client{
		baseURL:    fmt.Sprintf("https://dev.azure.com/%s/%s/_apis", cfg.Org, cfg.Project),
		orgURL:     orgURL,
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
	encoded := base64.StdEncoding.EncodeToString([]byte(":" + c.token))
	return "Basic " + encoded
}

// doRequest executes an HTTP request with auth, api-version, and 401 retry.
func (c *Client) doRequest(method, fullURL string, body interface{}, result interface{}) error {
	if err := c.ensureToken(); err != nil {
		return err
	}

	// Append api-version
	if strings.Contains(fullURL, "?") {
		fullURL += "&api-version=7.1"
	} else {
		fullURL += "?api-version=7.1"
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
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if c.authMethod == config.AuthAzCLI {
			c.mu.Lock()
			err := c.refreshAzCLIToken()
			c.mu.Unlock()
			if err != nil {
				return fmt.Errorf("token refresh failed: %w", err)
			}
			// Rebuild request for retry (body may have been consumed)
			if body != nil {
				jsonBytes, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(jsonBytes)
			}
			req, _ = http.NewRequest(method, fullURL, bodyReader)
			req.Header.Set("Authorization", c.authHeader())
			req.Header.Set("Accept", "application/json")
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err = c.httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("retry request failed: %w", err)
			}
			defer resp.Body.Close()
		} else {
			return fmt.Errorf("authentication failed — check that your PAT is valid and has not expired")
		}
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
