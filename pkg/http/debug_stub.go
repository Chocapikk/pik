//go:build !debug

package http

import nethttp "net/http"

func debugRequest(_ *nethttp.Request, _ []byte)  {}
func debugResponse(_ *nethttp.Response)           {}
