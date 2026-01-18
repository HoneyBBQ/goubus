// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package log

// Log represents system log entries.
type Log struct {
	Log []Data `json:"log"`
}

// Data represents a single log entry.
type Data struct {
	Text string `json:"text"`
	Time int    `json:"time"`
}
