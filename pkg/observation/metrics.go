package observation

import "sync/atomic"

// Metrics 轻量计数器。避免强依赖 prometheus，由宿主进程按需导出。
type Metrics struct {
	Recorded   atomic.Int64
	Dropped    atomic.Int64
	Failed     atomic.Int64
	GapMarkers atomic.Int64
}

// Snapshot 返回当前计数快照。
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"recorded":    m.Recorded.Load(),
		"dropped":     m.Dropped.Load(),
		"failed":      m.Failed.Load(),
		"gap_markers": m.GapMarkers.Load(),
	}
}
