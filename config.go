// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copyright (c) 2013 JamesClonk

package main

import (
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	ForceCompile    bool
	CompleteBuild   bool
	HotReload       bool
	GoplayDirectory string
}

var configRx = regexp.MustCompile(`\s*([[:alpha:]]\w*)\s+(.+)`)

// Read configuration and overwrite values if found
func ReadConfigurationFile(filename string, config *Config) bool {
	if Exist(filename) {
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalf("Could not read configuration file [%s]: %s", filename, err)
		}

		properties := make(map[string]string)
		if matched := configRx.FindAllStringSubmatch(string(bytes), -1); matched != nil {
			for _, match := range matched {
				// Convert to lowercase, and remove all underscores
				key := strings.Replace(strings.ToLower(match[1]), "_", "", -1)
				value := strings.ToLower(strings.Trim(match[2], "\t "))
				properties[key] = value
			}
		}

		if value, found := properties["forcecompile"]; found {
			flag, _ := strconv.ParseBool(value)
			config.ForceCompile = value == "yes" || flag
		}
		if value, found := properties["completebuild"]; found {
			flag, _ := strconv.ParseBool(value)
			config.CompleteBuild = value == "yes" || flag
		}
		if value, found := properties["hotreload"]; found {
			flag, _ := strconv.ParseBool(value)
			config.HotReload = value == "yes" || flag
		}
		if value, found := properties["goplaydirectory"]; found {
			config.GoplayDirectory = value
		}
		return true
	}

	return false
}
