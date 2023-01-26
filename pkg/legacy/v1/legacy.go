//go:generate ./fetch-legacy-plugin-index.sh

package v1

import (
	"embed"
)

//go:embed plugins plugins.json
var PluginIndex embed.FS
