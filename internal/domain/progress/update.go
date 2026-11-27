package progress

type Stage string

const (
	StageBackingUp   Stage = "Backing up"
	StageRestoring   Stage = "Restoring"
	StageVerifying   Stage = "Verifying"
	StageReadingData Stage = "Reading data"
	StageRekeying    Stage = "Rekeying"
	StageRetaining   Stage = "Extending retention"
)

func (s Stage) Unit() string {
	switch s {
	case StageBackingUp, StageRestoring, StageVerifying:
		return "files"
	default:
		return "objects"
	}
}

type Update struct {
	Stage      Stage
	Items      int
	TotalItems int
	Bytes      int64
	TotalBytes int64
	Finished   bool
}

func Started(stage Stage, totalItems int, totalBytes int64) Update {
	return Update{Stage: stage, TotalItems: totalItems, TotalBytes: totalBytes}
}

func (u Update) Advanced(items int, bytes int64) Update {
	u.Items += items
	u.Bytes += bytes

	return u
}

func (u Update) Completed() Update {
	u.Finished = true

	return u
}

func (u Update) Fraction() float64 {
	if u.Finished {
		return 1
	}

	switch {
	case u.TotalBytes > 0:
		return clamp(float64(u.Bytes) / float64(u.TotalBytes))
	case u.TotalItems > 0:
		return clamp(float64(u.Items) / float64(u.TotalItems))
	default:
		return 0
	}
}

func (u Update) MeasuresBytes() bool {
	return u.TotalBytes > 0
}

func clamp(value float64) float64 {
	return min(1, max(0, value))
}

type Listener interface {
	Report(update Update)
}
