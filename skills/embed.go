// Package skills embeds the sw skill files for distribution.
package skills

import "embed"

//go:embed sw
var Files embed.FS
