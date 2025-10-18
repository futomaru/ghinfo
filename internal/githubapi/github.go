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
	errorBodyReadLimit = 4 << 10 // エラー本文は4KBまでに制限して読み取る
	defaultTimeout     = 5 * time.Second
)

// Doer は http.Client と同じ Do メソッドを持つインターフェース。
// テスト時に HTTP 呼び出しを差し替えられるように定義している。
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client は GitHub API へアクセスするためのクライアントを表す。
// HTTP クライアントや API バージョン、トークンなどを保持する。
type Client struct {
	baseURL    string
	httpClient Doer
	token      string
	apiVersion string
}

// NewClient は標準の http.Client をベースにした Client を生成する。
// timeout が0以下の場合はデフォルトのタイムアウトを適用する。
func NewClient(timeout time.Duration, token string) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: timeout},
		token:      strings.TrimSpace(token),
		apiVersion: defaultAPIVersion,
	}
}

// SetHTTPClient は外部から Doer を差し替えるためのセッター。
// 主にテスト用に利用することを想定している。
func (c *Client) SetHTTPClient(doer Doer) {
	if doer == nil {
		return
	}
	c.httpClient = doer
}

// RepositoryInfo は CLI 側で利用しやすい形に整形したリポジトリ情報。
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

// apiRepository は GitHub API のレスポンス形式を受け取るための内部構造体。
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

// RepositoryInfoError は API からのエラーレスポンスを表すカスタムエラー。
type RepositoryInfoError struct {
	StatusCode int
	Message    string
}

func (e *RepositoryInfoError) Error() string {
	return fmt.Sprintf("github api error: status=%d message=%s", e.StatusCode, e.Message)
}

// GetRepository は owner/repo 形式の文字列を受け取り、リポジトリ情報を取得する。
func (c *Client) GetRepository(ctx context.Context, ownerRepo string) (RepositoryInfo, error) {
	var info RepositoryInfo

	if c.httpClient == nil {
		return info, errors.New("http client is not configured")
	}

	ownerRepo = strings.TrimSpace(ownerRepo)
	if ownerRepo == "" {
		return info, errors.New("repository identifier is empty")
	}

	if !strings.Contains(ownerRepo, "/") {
		return info, fmt.Errorf("repository identifier must be in the form owner/repo: %s", ownerRepo)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/repos/"+ownerRepo, nil)
	if err != nil {
		return info, fmt.Errorf("failed to build request: %w", err)
	}

	// GitHub API が推奨するヘッダ群をセットする。
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("X-GitHub-Api-Version", c.apiVersion)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return info, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, errorBodyReadLimit))
		if readErr != nil {
			return info, &RepositoryInfoError{StatusCode: resp.StatusCode, Message: resp.Status}
		}
		return info, &RepositoryInfoError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	var apiResp apiRepository
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return info, fmt.Errorf("failed to decode response: %w", err)
	}

	info = RepositoryInfo{
		FullName:    apiResp.FullName,
		Description: apiResp.Description,
		HTMLURL:     apiResp.HTMLURL,
		Language:    apiResp.Language,
		License:     extractLicenseName(apiResp.License),
		Stars:       apiResp.StargazersCount,
		Forks:       apiResp.ForksCount,
		Watchers:    apiResp.WatchersCount,
		OpenIssues:  apiResp.OpenIssuesCount,
		Archived:    apiResp.Archived,
		Disabled:    apiResp.Disabled,
		UpdatedAt:   apiResp.UpdatedAt,
		PushedAt:    apiResp.PushedAt,
	}

	return info, nil
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
