package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetOwnerRepo(t *testing.T) {
	owner, repo := getOwnerRepo("owner/repo")
	require.Equal(t, "owner", owner)
	require.Equal(t, "repo", repo)
}
