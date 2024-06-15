package pages

type JobWorker struct {
	ID                      string
	Queue                   string
	NotSeenSince            string
	Version                 string
	JobTypes                []string
	Workers                 int
	LastSeenAtColourSuccess bool
}
