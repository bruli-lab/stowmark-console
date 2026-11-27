package retention

import "time"

type Decision struct {
	SnapshotID string
	CreatedAt  time.Time
	Source     string
	Reasons    []string
}

func (d *Decision) Keep() bool {
	return len(d.Reasons) > 0
}

type Plan []Decision

func (p Plan) Forgotten() int {
	forgotten := 0
	for i := range p {
		if !p[i].Keep() {
			forgotten++
		}
	}

	return forgotten
}

func (p Plan) Kept() int {
	return len(p) - p.Forgotten()
}
