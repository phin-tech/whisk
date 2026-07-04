package terminal

type Options struct {
	MaxScrollbackLines       int
	MaxSnapshotFieldBytes    int
	MaxTitleBytes            int
	MaxWorkingDirectoryBytes int
}

type Cursor struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Snapshot struct {
	Offset                  uint64              `json:"offset"`
	Cols                    int                 `json:"cols"`
	Rows                    int                 `json:"rows"`
	Cursor                  Cursor              `json:"cursor"`
	Title                   string              `json:"title,omitempty"`
	WorkingDirectory        string              `json:"workingDirectory,omitempty"`
	ScrollbackAnsi          string              `json:"scrollbackAnsi"`
	RehydrateBeforeViewport string              `json:"rehydrateBeforeViewport,omitempty"`
	ViewportAnsi            string              `json:"viewportAnsi"`
	RehydrateSequences      string              `json:"rehydrateSequences"`
	Modes                   Modes               `json:"modes"`
	MouseTrackingModes      []MouseTrackingMode `json:"mouseTrackingModes,omitempty"`
	MouseEncodingModes      []MouseEncodingMode `json:"mouseEncodingModes,omitempty"`
	Truncated               bool                `json:"truncated,omitempty"`
}

const (
	defaultCols                    = 80
	defaultRows                    = 24
	defaultMaxScrollbackLines      = 10000
	defaultMaxSnapshotFieldBytes   = 4 * 1024 * 1024
	defaultMaxTitleBytes           = 1024
	defaultMaxWorkingDirectoryByte = 4096
	maxCols                        = 1000
	maxRows                        = 1000
)

func normalizeOptions(opts Options) Options {
	if opts.MaxScrollbackLines <= 0 {
		opts.MaxScrollbackLines = defaultMaxScrollbackLines
	}
	if opts.MaxSnapshotFieldBytes <= 0 {
		opts.MaxSnapshotFieldBytes = defaultMaxSnapshotFieldBytes
	}
	if opts.MaxTitleBytes <= 0 {
		opts.MaxTitleBytes = defaultMaxTitleBytes
	}
	if opts.MaxWorkingDirectoryBytes <= 0 {
		opts.MaxWorkingDirectoryBytes = defaultMaxWorkingDirectoryByte
	}
	return opts
}

func normalizeSize(cols, rows int) (int, int) {
	if cols <= 0 {
		cols = defaultCols
	}
	if rows <= 0 {
		rows = defaultRows
	}
	if cols > maxCols {
		cols = maxCols
	}
	if rows > maxRows {
		rows = maxRows
	}
	return cols, rows
}
