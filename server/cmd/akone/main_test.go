package main

import "testing"

func TestGlobalConfigPath(t *testing.T) {
	path, args, err := globalConfigPath([]string{"--config", "/tmp/akone.yml", "version", "--json"})
	if err != nil || path != "/tmp/akone.yml" || len(args) != 2 || args[0] != "version" {
		t.Fatalf("path=%q args=%v err=%v", path, args, err)
	}
	if _, _, err = globalConfigPath([]string{"--config"}); err == nil {
		t.Fatal("missing configuration path was accepted")
	}
}
