// Package modules imports all exploit modules and C2 backends
// so they self-register via init().
//
//	_ "github.com/Chocapikk/pik/modules"
package modules

import (
	// Exploit modules — Linux
	_ "github.com/Chocapikk/pik/modules/exploit/linux/http"
	_ "github.com/Chocapikk/pik/modules/exploit/linux/tcp"

	// Exploit modules — Multi-platform
	_ "github.com/Chocapikk/pik/modules/exploit/multi/http"

	// C2 backends
	_ "github.com/Chocapikk/pik/pkg/c2/httpshell"
	_ "github.com/Chocapikk/pik/pkg/c2/sliver"
	_ "github.com/Chocapikk/pik/pkg/c2/sslshell"
)
