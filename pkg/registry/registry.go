package registry

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type Plugin struct {
	FullName      string
	Type          string
	Name          string
	URL           string
	LatestRelease *PluginRelease
	Versions      []string
}

type PluginRelease struct {
	Version    string
	Prerelease bool
	CreatedAt  time.Time
	Assets     map[string]*PluginAsset
}

type PluginAsset struct {
	FileName string
	URL      string
	OS       string
	Arch     string
	Checksum string
}

type BatchRequestPlugin struct {
	FullName          string
	VersionConstraint string
}

type BatchRequest struct {
	OS      string
	Arch    string
	Plugins []*BatchRequestPlugin
}

func (b *BatchRequest) GetOSArch() string {
	return fmt.Sprintf("%s/%s", b.OS, b.Arch)
}

func (b *BatchRequest) Validate() error {
	if b.OS == "" || b.Arch == "" {
		return fmt.Errorf("os and arch are required")
	}

	if len(b.Plugins) == 0 {
		return fmt.Errorf("at least one plugin is required")
	}

	if len(b.Plugins) > 10 {
		return fmt.Errorf("maximum of 10 plugins allowed")
	}
	return nil
}

type BatchResponsePlugin struct {
	*BatchRequestPlugin
	Version  string
	FileName string
	URL      string
	Checksum string
}

func NewBatchResponsePlugin(req *BatchRequestPlugin) *BatchResponsePlugin {
	return &BatchResponsePlugin{
		BatchRequestPlugin: &BatchRequestPlugin{
			FullName:          strings.ToLower(req.FullName),
			VersionConstraint: req.VersionConstraint,
		},
	}
}

func (b *BatchResponsePlugin) String() string {
	return fmt.Sprintf("%s@%s (version=%s) (checksum=%s)", b.FullName, b.VersionConstraint, b.Version, b.Checksum)
}

func (b *BatchResponsePlugin) Hash() []byte {
	h := sha512.New512_256()
	_, _ = io.WriteString(h, b.String())
	return h.Sum(nil)
}

type BatchResponsePlugins []*BatchResponsePlugin

func (b BatchResponsePlugins) Len() int {
	return len(b)
}

func (b BatchResponsePlugins) Less(i, j int) bool {
	return strings.Compare(b[i].FullName, b[j].FullName) < 0
}

func (b BatchResponsePlugins) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b BatchResponsePlugins) Hash() []byte {
	h := sha512.New512_256()
	for _, c := range b {
		_, _ = h.Write(c.Hash())
	}
	return h.Sum(nil)
}

func (b BatchResponsePlugins) Has(fullName string) bool {
	for _, c := range b {
		if c.FullName == strings.ToLower(fullName) {
			return true
		}
	}
	return false
}

type BatchResponse struct {
	OS           string
	Arch         string
	Plugins      BatchResponsePlugins
	DownloadHash string
	DownloadURL  string
}

func NewBatchResponse(req *BatchRequest, plugins BatchResponsePlugins) *BatchResponse {
	sort.Sort(plugins)
	return &BatchResponse{
		OS:      strings.ToLower(req.OS),
		Arch:    strings.ToLower(req.Arch),
		Plugins: plugins,
	}
}

func (b *BatchResponse) GetOSArch() string {
	return fmt.Sprintf("%s/%s", b.OS, b.Arch)
}

func (b *BatchResponse) Hash() []byte {
	h := sha512.New512_256()
	_, _ = io.WriteString(h, b.GetOSArch())
	_, _ = h.Write(b.Plugins.Hash())
	hSum := h.Sum(nil)
	b.DownloadHash = hex.EncodeToString(hSum)
	return hSum
}
