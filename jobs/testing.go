package jobs

import "testing"

// Test returns a TestQueue tuned for unit testing.
func Test(t *testing.T) *TestQueue {
	if t == nil {
		panic("t is nil")
	}

	queue := newMemoryQueue()

	return &TestQueue{
		Queue: queue,
		TestAssertions: &TestAssertions{
			queue: queue,
			t:     t,
		},
	}
}

// TestQueue is a special Queue for unit testing.
// It exposes all methods of Queue and can be injected as a dependency
// in any application.
// Additionally, TestQueue exposes a set of assertions TestAssertions
// on all the jobs stored in the Queue.
type TestQueue struct {
	Queue
	*TestAssertions
}

// TestAssertions are assertions that work on a Queue, to make
// testing easier and convenient.
// The interface follows stretchr/testify as close as possible.
//
//   - Every assert func returns a bool indicating whether the assertion was successful or not,
//     this is useful for if you want to go on making further assertions under certain conditions.
type TestAssertions struct {
	queue Queue
	t     *testing.T
}
