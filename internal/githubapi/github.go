package githubapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL     = "https://api.github.com"
	defaultAPIVersion  = "2022-11-28"
	acceptHeader       = "application/vnd.github+json"
	errorBodyReadLimit = 4 << 10 // エラー本文は 4KB まで読めれば十分とする
	defaultTimeout     = 5 * time.Second
)

// Client は GitHub REST API への問い合わせを担当する。
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	apiVersion string
}

// NewClient はデフォルト設定のクライアントを返す。
func NewClient(timeout time.Duration, token string) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
		token:      strings.TrimSpace(token),
		apiVersion: defaultAPIVersion,
	}
}

// SetHTTPClient は外部で用意した http.Client を差し替えたい場合に利用する。
func (c *Client) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	c.httpClient = client
}

// RepositoryInfo は CLI へ渡すためのリポジトリ情報。
type RepositoryInfo struct {
	FullName    string
	Description string
	HTMLURL     string
	Language    string
	License     string
	Stars       int
	Forks       int
	Watchers    int
	OpenIssues  int
	Archived    bool
	Disabled    bool
	UpdatedAt   time.Time
	PushedAt    time.Time
}

// RepositoryInfoError は GitHub API からのエラー応答を保持する。
type RepositoryInfoError struct {
	StatusCode int
	Message    string
}

func (e *RepositoryInfoError) Error() string {
	return fmt.Sprintf("github api error: status=%d message=%s", e.StatusCode, e.Message)
}

// GetRepositoryByFullName は "owner/repo" 形式の入力を受け取る。
func (c *Client) GetRepositoryByFullName(ctx context.Context, fullName string) (RepositoryInfo, error) {
	owner, repo, err := splitFullName(fullName)
	if err != nil {
		return RepositoryInfo{}, err
	}
	return c.GetRepository(ctx, owner, repo)
}

// GetRepository は owner と repo を明示的に受け取って問い合わせる。
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (RepositoryInfo, error) {
	if c.httpClient == nil {
		return RepositoryInfo{}, errors.New("http client is not configured")
	}

	req, err := c.buildRequest(ctx, owner, repo)
	if err != nil {
		return RepositoryInfo{}, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if err := checkStatus(res); err != nil {
		return RepositoryInfo{}, err
	}

	return parseRepository(res.Body)
}

func (c *Client) buildRequest(ctx context.Context, owner, repo string) (*http.Request, error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return nil, errors.New("owner and repo must be specified")
	}

	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("X-GitHub-Api-Version", c.apiVersion)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return req, nil
}

func checkStatus(res *http.Response) error {
	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, errorBodyReadLimit))
	if err != nil {
		return &RepositoryInfoError{StatusCode: res.StatusCode, Message: res.Status}
	}

	return &RepositoryInfoError{
		StatusCode: res.StatusCode,
		Message:    strings.TrimSpace(string(body)),
	}
}

func parseRepository(r io.Reader) (RepositoryInfo, error) {
	var raw apiRepository
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return RepositoryInfo{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return RepositoryInfo{
		FullName:    raw.FullName,
		Description: raw.Description,
		HTMLURL:     raw.HTMLURL,
		Language:    raw.Language,
		License:     extractLicenseName(raw.License),
		Stars:       raw.StargazersCount,
		Forks:       raw.ForksCount,
		Watchers:    raw.WatchersCount,
		OpenIssues:  raw.OpenIssuesCount,
		Archived:    raw.Archived,
		Disabled:    raw.Disabled,
		UpdatedAt:   raw.UpdatedAt,
		PushedAt:    raw.PushedAt,
	}, nil
}

type apiRepository struct {
	FullName        string      `json:"full_name"`
	Description     string      `json:"description"`
	HTMLURL         string      `json:"html_url"`
	Language        string      `json:"language"`
	License         *apiLicense `json:"license"`
	StargazersCount int         `json:"stargazers_count"`
	ForksCount      int         `json:"forks_count"`
	WatchersCount   int         `json:"watchers_count"`
	OpenIssuesCount int         `json:"open_issues_count"`
	Archived        bool        `json:"archived"`
	Disabled        bool        `json:"disabled"`
	UpdatedAt       time.Time   `json:"updated_at"`
	PushedAt        time.Time   `json:"pushed_at"`
}

type apiLicense struct {
	Name   string `json:"name"`
	SpdxID string `json:"spdx_id"`
}

func extractLicenseName(license *apiLicense) string {
	if license == nil {
		return ""
	}
	if license.Name != "" {
		return license.Name
	}
	return license.SpdxID
}

func splitFullName(fullName string) (string, string, error) {
	fullName = strings.TrimSpace(fullName)
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be in the form owner/repo: %s", fullName)
	}
	return parts[0], parts[1], nil
}
