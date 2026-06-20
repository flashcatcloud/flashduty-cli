// Package skilldoc derives a structured description of the flashduty CLI's
// command tree (the "dump") and uses it to generate and validate the
// command-cards that document the CLI for an LLM operator.
//
// The dump is the single source of truth: it is built in-process from the
// live cobra tree (see Build), so it can never drift from the binary it
// describes. The generator turns a dump into per-domain factual fences; the
// validator checks every documented `fduty …` example against the same dump.
package skilldoc

// Dump is the structured snapshot of the CLI's command tree. It is the JSON
// contract shared between the dump oracle, the validator, and the generator.
type Dump struct {
	Commands []Command `json:"commands"`
}

// Command is one runnable leaf of the CLI tree.
type Command struct {
	Path  string `json:"path"`  // space-joined name chain below root, e.g. "status-page change-create"
	Group string `json:"group"` // first path segment, e.g. "status-page"
	Short string `json:"short"`
	// Use is cobra's raw Use string, e.g. "change-create <page-id>". cligen folds
	// a required *_id field into a positional argument and records it here as a
	// <placeholder>; that field is then supplied positionally, NOT via its
	// same-named --flag (passing the flag alone fails the Args check). Capturing
	// Use is what lets the generator render the correct positional invocation —
	// the bare Path alone (which strips the placeholder) cannot.
	Use     string `json:"use"`
	Long    string `json:"long"` // cligen's Request/Response field text (authoritative for enums + nested --data)
	Example string `json:"example"`
	Flags   []Flag `json:"flags"`
}

// Flag is one flag of a command, as exposed by pflag.
type Flag struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Usage    string `json:"usage"`
	Required bool   `json:"required"`
}
