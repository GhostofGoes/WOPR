// Package wopr embeds the licence and notice texts that ship with the binary, so
// `wopr --licenses` works for every install method, including `go install`.
package wopr

import _ "embed"

// License is the MIT licence for the code.
//
//go:embed LICENSE
var License string

// Notice covers the film text, third-party text and the non-affiliation statement.
//
//go:embed NOTICE.md
var Notice string

// ThirdPartyNotices holds the licences of every module linked into the binary.
// Regenerate it with `go run ./internal/tools/notices`.
//
//go:embed THIRD_PARTY_NOTICES.txt
var ThirdPartyNotices string
