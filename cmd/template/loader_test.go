package template

import (
	"testing"
)

func Test_loadEmbeddedConfigs(t *testing.T) {
	got := loadEmbeddedConfigs()

	if len(got) == 0 {
		t.Errorf("loadEmbeddedConfigs() = %v, expected at least one", len(got))
	}
}
