// Package migrations embeds SQL applied once per filename on ledger boot.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
