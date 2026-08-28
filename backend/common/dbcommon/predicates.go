package dbcommon

import "entgo.io/ent/dialect/sql"

// The predicate is only enforced if condition is true
func MaybePredicate(condition bool, pred func(*sql.Selector)) func(*sql.Selector) {
	if condition {
		return pred
	}
	return func(s *sql.Selector) {}
}
