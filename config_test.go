// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"testing"
)

func TestReadConfigurationFile(t *testing.T) {
	config = Config{false, true, false, ".goplay"}

	found := ReadConfigurationFile("config.rc", &config)
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
	expected := ".goplay/test"
	if config.GoplayDirectory != expected {
		t.Errorf("GoplayDirectory not as expected, was [%s], but should be [%s]", config.GoplayDirectory, expected)
	}
}
