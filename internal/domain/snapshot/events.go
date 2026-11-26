package snapshot

type Created struct {
	Snapshot Snapshot
}

type Verified struct {
	SnapshotID string
	Failures   []Failure
}

func (v Verified) Succeeded() bool {
	return len(v.Failures) == 0
}

type Restored struct {
	SnapshotID  string
	Destination string
	Failures    []Failure
}

func (r Restored) Succeeded() bool {
	return len(r.Failures) == 0
}

type FileExtracted struct {
	SnapshotID string
	Path       string
	Location   string
}
