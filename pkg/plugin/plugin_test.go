package plugin

import (
	"testing"

	"github.com/Masterminds/semver/v3"

	"github.com/stretchr/testify/require"
)

func TestFindMatchingVersion(t *testing.T) {
	testCases := []struct {
		inputVersions   []string
		inputConstraint string
		expectedVersion string
	}{
		{inputVersions: []string{"1.0.0", "1.1.0", "1.2.0"}, inputConstraint: "^1.0.0", expectedVersion: "1.2.0"},
		{inputVersions: []string{"1.0.0", "1.1.0", "1.2.0"}, inputConstraint: "~1.1.0", expectedVersion: "1.1.0"},
		{inputVersions: []string{"1.0.0", "1.1.0", "1.2.0"}, inputConstraint: "1", expectedVersion: "1.2.0"},
	}

	for _, testCase := range testCases {
		constraint, err := semver.NewConstraint(testCase.inputConstraint)
		require.NoError(t, err)
		actualVersion, err := findMatchingVersion(testCase.inputVersions, constraint)
		require.NoError(t, err)
		require.Equal(t, testCase.expectedVersion, actualVersion)
	}

	constraint, err := semver.NewConstraint("^3.0.0")
	require.NoError(t, err)
	_, err = findMatchingVersion([]string{"1.0.0", "1.1.0", "1.2.0"}, constraint)
	require.ErrorContains(t, err, "no matching version found")
}
