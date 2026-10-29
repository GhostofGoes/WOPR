// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux && !amd64 && !arm64

package sandbox

// Other architectures have no table, so TestSeccompNumbers skips; these keep the tests building.
var hostNumbers, hostAllowNumbers map[string]uintptr
