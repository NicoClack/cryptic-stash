// Avoid using this if at all possible, prefer storing state in services and storing them in a common.App.
// This allows state changes in tests to be independent of one another, which is usually what you want,
// except in a few weird situations

package globals

import "sync"

var (
	// Ent/Goose(?) has some internal state that prevents concurrent migrations on different databases
	MigrateMu sync.Mutex
)
