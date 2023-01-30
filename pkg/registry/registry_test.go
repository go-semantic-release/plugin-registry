package registry

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func newBatchPluginResponse(fullName, versionConstraint, version string) *BatchPluginResponse {
	res := NewBatchPluginResponse(&BatchPluginRequest{FullName: fullName, VersionConstraint: versionConstraint})
	res.Version = version
	return res
}

func TestBatchPluginHash(t *testing.T) {
	testCases := []struct {
		input    *BatchPluginResponse
		expected string
	}{
		{input: newBatchPluginResponse("foo", "^1.0.0", "1.2.3"), expected: "6e9c2ee756a18cfb7d4a01bc7863e5844a83f071b277dffd2dfa12e501e7fb0e"},
		{input: newBatchPluginResponse("Foo", "^1.0.0", "1.2.3"), expected: "6e9c2ee756a18cfb7d4a01bc7863e5844a83f071b277dffd2dfa12e501e7fb0e"},
	}

	for _, testCase := range testCases {
		actual := testCase.input.Hash()
		require.Equal(t, testCase.expected, hex.EncodeToString(actual))
	}
}

func TestBatchRequestHash(t *testing.T) {
	testCases := []struct {
		input    *BatchResponse
		expected string
	}{
		{
			input: &BatchResponse{
				BatchRequest: &BatchRequest{
					OS:   "darwin",
					Arch: "amd64",
				},
				Plugins: BatchPluginResponses{
					newBatchPluginResponse("foo", "^1.0.0", "1.2.3"),
					newBatchPluginResponse("bar", "^2.0.0", "2.2.3"),
				},
			},
			expected: "ab323e06aea1e43de11d5d272ab8d3d88375d934c5436d6d332e02f6223af0eb",
		},
	}

	for _, testCase := range testCases {
		res := NewBatchResponse(testCase.input.BatchRequest, testCase.input.Plugins)
		actual := res.Hash()
		require.Equal(t, "darwin", res.OS)
		require.Equal(t, "amd64", res.Arch)
		require.Equal(t, testCase.expected, hex.EncodeToString(actual))
	}
}
