package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
)

type (
	Release struct {
		URL         string    `json:"url"`
		HTMLURL     string    `json:"html_url"`
		TagName     string    `json:"tag_name"`
		Name        string    `json:"name"`
		Body        string    `json:"body"`
		Draft       bool      `json:"draft"`
		PreRelease  bool      `json:"prerelease"`
		Assets      []Asset   `json:"assets"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		PublishedAt time.Time `json:"published_at"`
		Author      User      `json:"author"`
	}
	Asset struct {
		URL                string `json:"url"`
		BrowserDownloadURL string `json:"browser_download_url"`
		ID                 int    `json:"id"`
		Name               string `json:"name"`
		Label              string `json:"label"`
		State              string `json:"state"`
		ContentType        string `json:"content_type"`
		Size               int    `json:"size"`
		Digest             string `json:"digest"`
		DownloadCount      int    `json:"download_count"`
		Uploader           User   `json:"uploader"`
	}
	User struct {
		Login        string `json:"login"`
		ID           int    `json:"id"`
		URL          string `json:"url"`
		HTMLURL      string `json:"html_url"`
		Type         string `json:"type"`
		UserViewType string `json:"user_view_type"`
	}
)

func getCurrentModule() (moduleName string, err error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("build info not found")
	}

	moduleName = buildInfo.Main.Path
	if !strings.HasPrefix(moduleName, "github.com") {
		err = fmt.Errorf("just github.com repositories are accepted by now - %s", moduleName)
	}

	return moduleName, err
}

func GetThisLatestRelease() (*Release, error) {
	moduleName, err := getCurrentModule()
	if err != nil {
		return nil, err
	}
	// Extract owner and repo from github URL
	url, err := url.Parse(moduleName)
	if err != nil {
		return nil, err
	}

	words := strings.Split(url.Path, "/")
	if len(words) < 3 {
		return nil, fmt.Errorf("invalid url %s", url)
	}

	owner, repo := words[1], words[2]

	return GetLatestRelease(owner, repo)
}

var githubClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		if req.URL.Host != "api.github.com" && req.URL.Host != "github.com" {
			return fmt.Errorf("redirect to untrusted host: %s", req.URL.Host)
		}
		return nil
	},
}

func GetLatestRelease(owner, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("X-Github-Api-Version", "2026-03-10")
	req.Header.Add("Accept", "application/vnd.github+json")

	response, err := githubClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}
	defer response.Body.Close()

	var release Release
	if err = json.NewDecoder(response.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed deserialization of release: %w", err)
	}

	return &release, nil
}

func (asset *Asset) Download(w io.Writer) error {
	resp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	chash := digest.FromBytes(content)
	if chash.String() != asset.Digest {
		return fmt.Errorf("asset downloaded from %s does not match digest %s", asset.BrowserDownloadURL, asset.Digest)
	}

	_, err = w.Write(content)

	return err
}
