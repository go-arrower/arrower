package models

// DB is manually added as a shortcut to support squirrel access to the database.
// See Register implementation for more context.
func (q *Queries) DB() DBTX {
	return q.db
}
