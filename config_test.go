// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestReadConfigurationFile(t *testing.T) {
	config = Config{false, true, false, false, []string{"go"}, ".goplay"}

	found := ReadConfigurationFile("config/config.rc", &config)
	if !found {
		t.Error("Configuration file could not be found, but it should be")
	}

	if !config.ForceCompile {
		t.Error("ForceCompile should now be set to 'true', but it is not")
	}
	if config.CompleteBuild {
		t.Error("CompleteBuild should now be set to 'false', but it is not")
	}
	if !config.HotReload {
		t.Error("HotReload should now be set to 'true', but it is not")
	}
	if !config.HotReloadRecursive {
		t.Error("HotReloadRecursive should now be set to 'true', but it is not")
	}
	expectedExtensions := []string{"go", "html"}
	for _, extension := range expectedExtensions {
		if !config.HotReloadWatchExtensions.Contains(extension) {
			t.Errorf("HotReloadWatchExtensions not as expected, was [%s], but should be [%s]", config.HotReloadWatchExtensions, expectedExtensions)
		}
	}
	expectedDirectory := ".goplay/test"
	if config.GoplayDirectory != expectedDirectory {
		t.Errorf("GoplayDirectory not as expected, was [%s], but should be [%s]", config.GoplayDirectory, expectedDirectory)
	}
}

func TestLocalGoplayRc(t *testing.T) {
	out, err := exec.Command("./config/config.go").Output()
	if err != nil {
		t.Fatal(err)
	}
	expected(t, "config/config.go", string(out), "local .goplayrc test\n")

	if !Exist("config/.config_test") {
		t.Errorf("Directory does not exist: [%s] ", ".config_test")
	}

	if err := os.RemoveAll("config/.config_test"); err != nil {
		t.Fatal(err)
	}
}
