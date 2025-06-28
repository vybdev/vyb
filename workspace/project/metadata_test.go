package project

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMetadata_Patch(t *testing.T) {
	type testCase struct {
		name              string
		stored            *Metadata
		fresh             *Metadata
		expected          *PatchResult
		expectedErr       string
		expectedModules   []string
		expectedFiles     []string
		expectedRemoved   []string
		expectedAdded     []string
		expectedChanged   []string
		expectedUnchanged []string
	}

	testCases := []testCase{
		{
			name: "should detect no changes",
			stored: &Metadata{
				Modules: &Module{
					Name: ".",
					MD5:  "abc",
				},
			},
			fresh: &Metadata{
				Modules: &Module{
					Name: ".",
					MD5:  "abc",
				},
			},
			expected: &PatchResult{
				ChangedModules: map[string]ModuleChange{},
			},
		},
		{
			name: "should detect module changes",
			stored: &Metadata{
				Modules: &Module{
					Name:       ".",
					MD5:        "abc",
					TokenCount: 100,
				},
			},
			fresh: &Metadata{
				Modules: &Module{
					Name:       ".",
					MD5:        "def",
					TokenCount: 200,
				},
			},
			expected: &PatchResult{
				ChangedModules: map[string]ModuleChange{
					".": {
						PreviousTokenCount: 100,
						CurrentTokenCount:  200,
					},
				},
			},
		},
		{
			name: "should detect added modules",
			stored: &Metadata{
				Modules: &Module{
					Name: ".",
					MD5:  "abc",
				},
			},
			fresh: &Metadata{
				Modules: &Module{
					Name: ".",
					MD5:  "abc",
					Modules: []*Module{
						{
							Name: "new",
							MD5:  "def",
						},
					},
				},
			},
			expected: &PatchResult{
				ChangedModules: map[string]ModuleChange{},
				AddedModules:   []string{"new"},
			},
		},
		{
			name: "should detect removed modules",
			stored: &Metadata{
				Modules: &Module{
					Name: ".",
					MD5:  "abc",
					Modules: []*Module{
						{
							Name: "removed",
							MD5:  "def",
						},
					},
				},
			},
			fresh: &Metadata{
				Modules: &Module{
					Name: ".",
					MD5:  "abc",
				},
			},
			expected: &PatchResult{
				ChangedModules: map[string]ModuleChange{},
				RemovedModules: []string{"removed"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.stored.Patch(tc.fresh)
			assert.Equal(t, tc.expected, result)
		})
	}
}