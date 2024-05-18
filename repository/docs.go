// Package repository panics instead of returning errors,
// to make calling it much easier and prevent boilerplate in error checking.
// The repositories are intended for rapid prototyping and development and not production use.
//
// A Repository offers a whole set of methods already out of the box. That might not be enough, though.
// It is possible to overwrite an existing method to change the behaviour as well as extend the Repository
// with new methods. There are examples for both.
//
// The primary use case is for testing and quick iterations when prototyping, so all repositories are in memory.
// Sometimes it might be handy so persist some data, so it is possible to use a Store to do so.
// This is NOT intended for production use and only recommended for local demoing of an application.
// With the application worked out, it's best to implement a proper repository that stores the data in a real datastore.
package repository
