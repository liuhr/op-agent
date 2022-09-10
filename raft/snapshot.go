package oraft

import (
	"io"
)

type SnapshotCreatorApplier interface {
	GetData() (data []byte, err error)
	Restore(rc io.ReadCloser) error
}
