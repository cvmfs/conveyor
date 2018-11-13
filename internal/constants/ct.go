package constants

const (
	// NewJobExchange - name of RabbitMQ exchange for new jobs
	NewJobExchange string = "jobs.new"
	// NewJobQueue - name of the RabbitMQ queue for new jobs
	NewJobQueue string = "jobs.new"
	// RoutingKey - routing key used when publishing jobs
	RoutingKey string = ""
)
