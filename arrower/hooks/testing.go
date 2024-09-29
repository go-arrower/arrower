package hooks

import "testing"

/*
Use this to test your plugin implementation:
* Can be tested bare bones
* This runs it through yaegi => 100% same code as in run cli
* Load individual file or whole folder (detects automatically, if the path is a file or folder)
*/

func Test(t *testing.T) []Hook {
	_ = t
	return nil
}
