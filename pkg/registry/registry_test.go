package registry

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestBatchResponsePlugin(fullName, versionConstraint, version string) *BatchResponsePlugin {
	res := NewBatchResponsePlugin(&BatchRequestPlugin{FullName: fullName, VersionConstraint: versionConstraint})
	res.Version = version
	return res
}

func TestBatchPluginHash(t *testing.T) {
	testCases := []struct {
		input    *BatchResponsePlugin
		expected string
	}{
		{input: newTestBatchResponsePlugin("foo", "^1.0.0", "1.2.3"), expected: "6e9c2ee756a18cfb7d4a01bc7863e5844a83f071b277dffd2dfa12e501e7fb0e"},
		{input: newTestBatchResponsePlugin("Foo", "^1.0.0", "1.2.3"), expected: "6e9c2ee756a18cfb7d4a01bc7863e5844a83f071b277dffd2dfa12e501e7fb0e"},
	}

	for _, testCase := range testCases {
		actual := testCase.input.Hash()
		require.Equal(t, testCase.expected, hex.EncodeToString(actual))
	}
}

func TestBatchRequestHash(t *testing.T) {
	testCases := []struct {
		inputRequest *BatchRequest
		inputPlugins BatchResponsePlugins
		expected     string
	}{
		{
			inputRequest: &BatchRequest{
				OS:   "darwin",
				Arch: "amd64",
			},
			inputPlugins: BatchResponsePlugins{
				newTestBatchResponsePlugin("foo", "^1.0.0", "1.2.3"),
				newTestBatchResponsePlugin("bar", "^2.0.0", "2.2.3"),
			},
			expected: "ab323e06aea1e43de11d5d272ab8d3d88375d934c5436d6d332e02f6223af0eb",
		},
	}

	for _, testCase := range testCases {
		res := NewBatchResponse(testCase.inputRequest, testCase.inputPlugins)
		actual := res.Hash()
		require.Equal(t, "darwin", res.OS)
		require.Equal(t, "amd64", res.Arch)
		require.Equal(t, testCase.expected, hex.EncodeToString(actual))
	}
}
