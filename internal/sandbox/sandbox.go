// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Package sandbox gives up what wopr no longer needs once the terminal is set up
// (docs/PLAN.md §8, "Process hardening"). On Linux it drops any capabilities, sets
// no_new_privs, and enters a Landlock domain with no rules: the process can no longer open,
// create or run any file, nor bind or connect a TCP socket. Files it opened before, the
// terminal and the debug log, stay usable. On other systems, and where the kernel lacks a
// feature, Restrict does what it can and says what it did; it never stops wopr.
package sandbox
