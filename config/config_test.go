package config

import (
    "path/filepath"
    "testing"
    "testing/fstest"
)

func TestLoadFS_Default(t *testing.T) {
    fsys := fstest.MapFS{}

    cfg, err := LoadFS(fsys)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.Provider != "openai" {
        t.Fatalf("expected default provider 'openai', got %s", cfg.Provider)
    }
}

func TestLoadFS_FromFile(t *testing.T) {
    fsys := fstest.MapFS{
        filepath.ToSlash(".vyb/config.yaml"): &fstest.MapFile{Data: []byte("provider: fooai\n")},
    }

    cfg, err := LoadFS(fsys)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.Provider != "fooai" {
        t.Fatalf("expected provider 'fooai', got %s", cfg.Provider)
    }
}
