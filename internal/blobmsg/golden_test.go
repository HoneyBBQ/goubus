// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package blobmsg_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

func TestGoldenFiles(t *testing.T) {
	// 1. Test real snapshots from hardware
	t.Run("Snapshots", func(t *testing.T) {
		files, _ := filepath.Glob("testdata/*.bin")
		for _, file := range files {
			t.Run(filepath.Base(file), func(t *testing.T) {
				data, err := os.ReadFile(filepath.Clean(file))
				if err != nil {
					t.Fatal(err)
				}

				attrs, err := blobmsg.ParseTopLevelAttributes(data)
				if err != nil {
					t.Errorf("Failed to parse %s: %v", file, err)
				}

				t.Logf("Parsed %d attributes from %s", len(attrs), file)
			})
		}
	})

	// 2. Test fuzzing corpus from official ubus
	t.Run("FuzzCorpus", func(t *testing.T) {
		files, _ := filepath.Glob("testdata/fuzz_corpus/*")
		for _, file := range files {
			t.Run(filepath.Base(file), func(t *testing.T) {
				data, err := os.ReadFile(filepath.Clean(file))
				if err != nil {
					t.Fatal(err)
				}
				// Many of these are malformed, we just want to ensure we don't PANIC
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PANIC while parsing %s: %v", file, r)
					}
				}()

				// Note: some of these might be raw blobmsg, some might be full ubus messages
				// We try to parse as top-level attributes first
				_, _ = blobmsg.ParseTopLevelAttributes(data)
				// Also try to parse as a nested container if it looks like one
				if len(data) > 4 {
					_, _ = blobmsg.ParseBlobmsgContainer(data[4:], blobmsg.TypeTable)
				}
			})
		}
	})
}
