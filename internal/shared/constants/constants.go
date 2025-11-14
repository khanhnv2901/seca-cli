package constants

import (
	"io/fs"
	"time"
)

const (
	// DefaultDirPerm is the default permission used when creating directories.
	DefaultDirPerm fs.FileMode = 0o755
	// DefaultFilePerm is the default permission used when creating files.
	DefaultFilePerm fs.FileMode = 0o644
)

const (
	// RawCaptureLimitBytes caps how many bytes of a response body we store for auditing.
	RawCaptureLimitBytes = 2048
	// TLSSoonExpiryWindow warns operators when a certificate expires inside this window.
	TLSSoonExpiryWindow = 14 * 24 * time.Hour
)
