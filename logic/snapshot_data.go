package logic

import (
	//"bytes"
	"io"
)

type SnapshotDataCreatorApplier struct {
}

func NewSnapshotDataCreatorApplier() *SnapshotDataCreatorApplier {
	generator := &SnapshotDataCreatorApplier{}
	return generator
}

func (this *SnapshotDataCreatorApplier) GetData() (data []byte, err error) {
	return data, nil
}

func (this *SnapshotDataCreatorApplier) Restore(rc io.ReadCloser) error {
	return nil
}
