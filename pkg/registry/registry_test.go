package registry

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func newBatchPluginResponse(fullName, versionConstraint, version string) *BatchPluginResponse {
	return &BatchPluginResponse{BatchPluginRequest: &BatchPluginRequest{FullName: fullName, VersionConstraint: versionConstraint}, Version: version}
}

func TestBatchPluginHash(t *testing.T) {
	testCases := []struct {
		input    *BatchPluginResponse
		expected string
	}{
		{input: newBatchPluginResponse("foo", "^1.0.0", "1.2.3"), expected: "4f1781ea45f062354af59d6fbc4bd5390678621dfc0a223d8d87e7ade495315a"},
		{input: newBatchPluginResponse("Foo", "^1.0.0", "1.2.3"), expected: "4f1781ea45f062354af59d6fbc4bd5390678621dfc0a223d8d87e7ade495315a"},
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
			expected: "d36138edb269339e3c2117afc24d7f45afdaa48bc12797e0213776578ae1f395",
		},
	}

	for _, testCase := range testCases {
		actual := testCase.input.Hash()
		require.Equal(t, "darwin", testCase.input.OS)
		require.Equal(t, "amd64", testCase.input.Arch)
		require.Equal(t, testCase.expected, hex.EncodeToString(actual))
	}
}
