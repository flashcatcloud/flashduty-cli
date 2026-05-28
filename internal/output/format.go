package output

// Format selects how command output is serialized.
type Format int

const (
	// FormatTable renders human-readable aligned columns (the default).
	FormatTable Format = iota
	// FormatJSON renders pretty-printed JSON.
	FormatJSON
	// FormatTOON renders Token-Oriented Object Notation — a compact
	// serialization that drops the per-row repeated keys JSON emits for
	// uniform arrays, materially reducing token count for list output.
	FormatTOON
)

// Structured reports whether the format is a machine-readable dump (JSON or
// TOON) rather than the human table view. Table footers, detail views, and
// interactive prompts are suppressed when the output is structured.
func (f Format) Structured() bool { return f == FormatJSON || f == FormatTOON }
