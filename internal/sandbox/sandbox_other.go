// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build !linux

package sandbox

import "runtime"

// Restrict does nothing outside Linux. It returns a line for the debug log.
func Restrict() string { return "sandbox: none on " + runtime.GOOS }
