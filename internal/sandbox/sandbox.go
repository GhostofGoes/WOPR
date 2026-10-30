// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Package sandbox gives up what wopr no longer needs once the terminal is set up
// (docs/PLAN.md §8, "Process hardening"). On Linux it sets no_new_privs, drops any
// capabilities, enters a Landlock domain with no rules, and installs a seccomp filter. Landlock
// stops the process from opening, creating or changing any file, and from binding or
// connecting a TCP socket. The filter refuses the system calls that wopr never makes after it
// starts: running a program, starting a process, opening any socket, changing a file's
// metadata, tracing, signalling or rescheduling other processes, and most of the kernel's
// other interfaces. Files it opened before, the
// terminal and the debug log, stay usable. On other systems, and where the kernel lacks a
// feature, Restrict does what it can and says what it did; it never stops wopr.
package sandbox
