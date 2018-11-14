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
)

// Connection - encapsulates the AMQP connection and channel
type Connection struct {
	Conn *amqp.Connection
	Chan *amqp.Channel
}

// NewConnection - create a new connection to the job queue
func NewConnection(
	username string, password string, host string,
	vhost string, port int) (*Connection, error) {
	dialStr := createConnectionURL(username, password, host, vhost, port)
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
