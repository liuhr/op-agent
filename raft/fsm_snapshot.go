package oraft

import (
	"github.com/hashicorp/raft"
)

// fsmSnapshot handles raft persisting of snapshots
type fsmSnapshot struct {
	snapshotCreatorApplier SnapshotCreatorApplier
}

func newFsmSnapshot(snapshotCreatorApplier SnapshotCreatorApplier) *fsmSnapshot {
	return &fsmSnapshot{
		snapshotCreatorApplier: snapshotCreatorApplier,
	}
}

// Persist
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	data, err := f.snapshotCreatorApplier.GetData()
	if err != nil {
		return err
	}
	if _, err := sink.Write(data); err != nil {
		return err
	}
	return sink.Close()
}

// Release
func (f *fsmSnapshot) Release() {
}
