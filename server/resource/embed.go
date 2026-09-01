// Package resource contains the only production source of public web templates.
package resource

import "embed"

//go:embed tpl static
var Files embed.FS
