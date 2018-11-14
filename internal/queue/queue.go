package queue

import (
	"strconv"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/streadway/amqp"
)

const (
	// NewJobExchange - name of RabbitMQ exchange for new jobs
	NewJobExchange string = "jobs.new"
	// NewJobQueue - name of the RabbitMQ queue for new jobs
	NewJobQueue string = "jobs.new"
	// RoutingKey - routing key used when publishing jobs
	RoutingKey string = ""
	// ConsumerName - name to identify the consumer
	ConsumerName string = "cvmfs_job"
)

// Parameters - connection parameters for the job queue
type Parameters struct {
	Username string
	Password string
	Host     string
	VHost    string
	Port     int
}

// Connection - encapsulates the AMQP connection and channel
type Connection struct {
	Conn *amqp.Connection
	Chan *amqp.Channel
}

// NewConnection - create a new connection to the job queue
func NewConnection(
	params Parameters) (*Connection, error) {
	dialStr := createConnectionURL(
		params.Username, params.Password, params.Host,
		params.VHost, params.Port)
	connection, err := amqp.Dial(dialStr)
	if err != nil {
		log.Error.Println("Could not open AMQP connection:", err)
		return nil, err
	}

	channel, err := connection.Channel()
	if err != nil {
		log.Error.Println("Could not open AMQP channel:", err)
		return nil, err
	}

	return &Connection{connection, channel}, nil
}

// Close - closes an established connection to the job queue
func (c *Connection) Close() error {
	return c.Conn.Close()
}

func createConnectionURL(username string,
	password string, host string, vhost string, port int) string {

	return "amqp://" + username +
		":" + password +
		"@" + host +
		":" + strconv.Itoa(port) + "/" + vhost
}
