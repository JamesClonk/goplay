// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"io/ioutil"
	"log"
	"strings"
)

type Config struct {
	ForceCompile    bool
	CompleteBuild   bool
	HotReload       bool
	GoplayDirectory string
}

// Read configuration and overwrite values if found
func ReadConfigurationFile(filename string, config *Config) bool {
	if Exist(filename) {
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalf("Could not read configuration file [%s]: %s", filename, err)
		}

		properties := make(map[string]string)

		lines := strings.Split(string(bytes), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// Convert to lowercase, and remove all underscores for "key"
				properties[strings.Replace(strings.ToLower(fields[0]), "_", "", -1)] = strings.ToLower(fields[1])
			}
		}

		if value, found := properties["forcecompile"]; found {
			config.ForceCompile = value == "yes"
		}
		if value, found := properties["completebuild"]; found {
			config.CompleteBuild = value == "yes"
		}
		if value, found := properties["hotreload"]; found {
			config.HotReload = value == "yes"
		}
		if value, found := properties["goplaydirectory"]; found {
			config.GoplayDirectory = value
		}
		return true
	}

	return false
}
