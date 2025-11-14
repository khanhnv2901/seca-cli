// Package constants centralizes configuration defaults shared across the CLI.
//
// Storing file permissions, raw-capture limits, and TLS warning windows in one
// place prevents magic numbers from scattering across cmd/ and internal/.
// The values here reflect conservative defaults that can be referenced from
// multiple packages without introducing import cycles.
package constants

