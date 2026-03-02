// Package modules imports all exploit modules and C2 backends
// so they self-register via init().
//
//	_ "github.com/Chocapikk/pik/modules"
package modules

import (
	// Exploit modules — HTTP
	_ "github.com/Chocapikk/pik/modules/exploit/http/linux"

	// C2 backends
	_ "github.com/Chocapikk/pik/pkg/c2/httpshell"
	_ "github.com/Chocapikk/pik/pkg/c2/sliver"
	_ "github.com/Chocapikk/pik/pkg/c2/sslshell"
)
