package plugin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/require"
)

func TestGetOwnerRepo(t *testing.T) {
	owner, repo := getOwnerRepo("owner/repo")
	require.Equal(t, "owner", owner)
	require.Equal(t, "repo", repo)
}

func TestGetAllGitHubReleases(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposReleasesByOwnerByRepo,
			[]*github.RepositoryRelease{
				{Draft: github.Bool(false), TagName: github.String("v1.0.0"), Assets: []*github.ReleaseAsset{{}}},
				{Draft: github.Bool(false), TagName: github.String("v1.0.1"), Assets: []*github.ReleaseAsset{{}}},
				{Draft: github.Bool(true), TagName: github.String("v2.0.0-beta"), Assets: []*github.ReleaseAsset{{}}},
				{Draft: github.Bool(false), TagName: github.String("v2.0.0"), Assets: []*github.ReleaseAsset{{}}},
			},
		),
	)
	ghClient := github.NewClient(mockedHTTPClient)
	ghReleases, err := getAllGitHubReleases(context.Background(), ghClient, "owner/repo")
	require.NoError(t, err)
	require.Len(t, ghReleases, 3)
	foundTags := make([]string, len(ghReleases))
	for i, ghRelease := range ghReleases {
		foundTags[i] = ghRelease.GetTagName()
	}
	require.ElementsMatch(t, []string{"v1.0.0", "v1.0.1", "v2.0.0"}, foundTags)
}

func TestGetGitHubRelease(t *testing.T) {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposReleasesTagsByOwnerByRepoByTag,
			&github.RepositoryRelease{Draft: github.Bool(false), TagName: github.String("v1.0.0"), Assets: []*github.ReleaseAsset{{}}},
			&github.RepositoryRelease{Draft: github.Bool(true), TagName: github.String("v2.0.0"), Assets: []*github.ReleaseAsset{{}}},
		),
	)
	ghClient := github.NewClient(mockedHTTPClient)
	ghRelease, err := getGitHubRelease(context.Background(), ghClient, "owner/repo", "v1.0.0")
	require.NoError(t, err)
	require.Equal(t, "v1.0.0", ghRelease.GetTagName())

	_, err = getGitHubRelease(context.Background(), ghClient, "owner/repo", "v2.0.0")
	require.ErrorContains(t, err, "release is a draft")
}

var testChecksumFile = `
0911f3dd  plugin_v1.0.0_windows_amd64.exe
0fe1a3ce  plugin_v1.0.0_darwin_amd64
50681c38  plugin_v1.0.0_darwin_arm64
8a491fb8  plugin_v1.0.0_linux_amd64
c3703969  plugin_v1.0.0_linux_arm
cacce75a  plugin_v1.0.0_linux_arm64
`

func getCheckSumServer(failingRequests int) *httptest.Server {
	cnt := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cnt++
		if cnt <= failingRequests {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, testChecksumFile)
	}))
}

func TestFetchChecksumFile(t *testing.T) {
	ts := getCheckSumServer(0)
	defer ts.Close()
	checksums, err := fetchChecksumFile(context.Background(), ts.URL)
	require.NoError(t, err)
	require.Len(t, checksums, 6)
	require.Equal(t, "0911f3dd", checksums["plugin_v1.0.0_windows_amd64.exe"])
	require.Equal(t, "cacce75a", checksums["plugin_v1.0.0_linux_arm64"])
}

func TestFetchChecksumFileRetry(t *testing.T) {
	ts := getCheckSumServer(1)
	defer ts.Close()
	checksums, err := fetchChecksumFile(context.Background(), ts.URL)
	require.NoError(t, err)
	require.Len(t, checksums, 6)
	require.Equal(t, "0911f3dd", checksums["plugin_v1.0.0_windows_amd64.exe"])
	require.Equal(t, "cacce75a", checksums["plugin_v1.0.0_linux_arm64"])
}

func TestGetPluginAssets(t *testing.T) {
	checksumServer := getCheckSumServer(0)
	defer checksumServer.Close()
	dlURL := github.String(checksumServer.URL)
	ghReleaseAssets := []*github.ReleaseAsset{
		{Name: github.String("plugin_v1.0.0_windows_amd64.exe"), Size: github.Int(123), BrowserDownloadURL: dlURL},
		{Name: github.String("plugin_v1.0.0_darwin_amd64"), Size: github.Int(234), BrowserDownloadURL: dlURL},
		{Name: github.String("plugin_v1.0.0_darwin_arm64"), Size: github.Int(345), BrowserDownloadURL: dlURL},
		{Name: github.String("plugin_v1.0.0_linux_amd64"), Size: github.Int(456), BrowserDownloadURL: dlURL},
		{Name: github.String("plugin_v1.0.0_linux_arm"), Size: github.Int(567), BrowserDownloadURL: dlURL},
		{Name: github.String("plugin_v1.0.0_linux_arm64"), Size: github.Int(456), BrowserDownloadURL: dlURL},
		{Name: github.String("checksums.txt"), Size: github.Int(789), BrowserDownloadURL: dlURL},
	}
	assets, err := getPluginAssets(context.Background(), ghReleaseAssets)
	require.NoError(t, err)
	require.Len(t, assets, 6)
	require.Equal(t, "0911f3dd", assets["windows/amd64"].Checksum)
	require.Equal(t, "50681c38", assets["darwin/arm64"].Checksum)
	require.Equal(t, "cacce75a", assets["linux/arm64"].Checksum)
}
