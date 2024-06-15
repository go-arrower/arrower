package jobs

const DefaultQueueName = QueueName("Default")

type (
	QueueName  string
	QueueNames []QueueName
)
