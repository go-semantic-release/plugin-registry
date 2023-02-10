package plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-semantic-release/plugin-registry/pkg/registry"
	"github.com/google/go-github/v50/github"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	defaultRetryableClient     *retryablehttp.Client
	defaultRetryableClientInit sync.Once
)

func getDefaultRetryableClient() *retryablehttp.Client {
	defaultRetryableClientInit.Do(func() {
		defaultRetryableClient = retryablehttp.NewClient()
		defaultRetryableClient.Logger = nil
		defaultRetryableClient.HTTPClient.Timeout = time.Minute
	})
	return defaultRetryableClient
}

func getOwnerRepo(fullRepo string) (string, string) {
	owner, repo, found := strings.Cut(fullRepo, "/")
	if !found {
		return "", ""
	}

	return owner, repo
}

func getAllGitHubReleases(ctx context.Context, ghClient *github.Client, fullRepo string) ([]*github.RepositoryRelease, error) {
	owner, repo := getOwnerRepo(fullRepo)
	ret := make([]*github.RepositoryRelease, 0)
	opts := &github.ListOptions{Page: 1, PerPage: 100}
	for {
		releases, resp, err := ghClient.Repositories.ListReleases(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		for _, release := range releases {
			// ignore drafts
			if release.GetDraft() {
				continue
			}
			// only include valid semver releases
			if _, err := semver.NewVersion(release.GetTagName()); err != nil {
				continue
			}

			// release has no assets attached
			if len(release.Assets) == 0 {
				continue
			}

			ret = append(ret, release)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return ret, nil
}

func getGitHubRelease(ctx context.Context, ghClient *github.Client, fullRepo, tag string) (*github.RepositoryRelease, error) {
	owner, repo := getOwnerRepo(fullRepo)
	release, _, err := ghClient.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return nil, err
	}
	if release.GetDraft() {
		return nil, fmt.Errorf("release is a draft")
	}
	if _, err := semver.NewVersion(release.GetTagName()); err != nil {
		return nil, fmt.Errorf("release is not a valid semver version: %w", err)
	}
	if len(release.Assets) == 0 {
		return nil, fmt.Errorf("release has no assets")
	}
	return release, nil
}

func fetchChecksumFile(ctx context.Context, url string) (map[string]string, error) {
	ret := make(map[string]string)
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := getDefaultRetryableClient().Do(req)
	if err != nil {
		return nil, err
	}
	checksums, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, err
	}
	for _, l := range strings.Split(string(checksums), "\n") {
		sl := strings.Split(l, " ")
		if len(sl) < 3 {
			continue
		}
		ret[strings.ToLower(sl[2])] = sl[0]
	}
	return ret, nil
}

var osArchRe = regexp.MustCompile(`(?i)(aix|android|darwin|dragonfly|freebsd|hurd|illumos|js|linux|nacl|netbsd|openbsd|plan9|solaris|windows|zos)(_|-)(386|amd64|amd64p32|arm|armbe|arm64|arm64be|ppc64|ppc64le|mips|mipsle|mips64|mips64le|mips64p32|mips64p32le|ppc|riscv|riscv64|s390|s390x|sparc|sparc64|wasm)(\.exe)?$`)

func getPluginAssets(ctx context.Context, gha []*github.ReleaseAsset) (map[string]*registry.PluginAsset, error) {
	assets := make([]*registry.PluginAsset, 0)
	var checksumMap map[string]string
	for _, asset := range gha {
		fn := asset.GetName()
		if checksumMap == nil && asset.GetSize() <= 4096 && strings.Contains(strings.ToLower(fn), "checksums.txt") {
			csMap, err := fetchChecksumFile(ctx, asset.GetBrowserDownloadURL())
			if err != nil {
				return nil, err
			}
			checksumMap = csMap
			continue
		}
		assets = append(assets, &registry.PluginAsset{
			FileName: fn,
			URL:      asset.GetBrowserDownloadURL(),
		})
	}

	ret := make(map[string]*registry.PluginAsset)
	for _, pa := range assets {
		osArch := osArchRe.FindAllStringSubmatch(pa.FileName, -1)
		if len(osArch) < 1 || len(osArch[0]) < 4 {
			continue
		}
		if checksumMap != nil {
			pa.Checksum = checksumMap[strings.ToLower(pa.FileName)]
		}
		os, arch := strings.ToLower(osArch[0][1]), strings.ToLower(osArch[0][3])
		pa.OS = os
		pa.Arch = arch
		ret[fmt.Sprintf("%s/%s", os, arch)] = pa
	}
	return ret, nil
}

func toPluginRelease(ctx context.Context, ghr *github.RepositoryRelease) (*registry.PluginRelease, error) {
	assets, err := getPluginAssets(ctx, ghr.Assets)
	if err != nil {
		return nil, err
	}

	return &registry.PluginRelease{
		Version:    semver.MustParse(ghr.GetTagName()).String(),
		Prerelease: ghr.GetPrerelease(),
		CreatedAt:  ghr.GetCreatedAt().Time,
		Assets:     assets,
	}, nil
}
