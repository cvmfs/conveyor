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

// Config - configuration of the job queue
type Config struct {
	Username string
	Password string
	Host     string
	VHost    string
	Port     int
}

// Connection - encapsulates the AMQP connection and channel
type Connection struct {
	Conn  *amqp.Connection
	Chan  *amqp.Channel
	Queue *amqp.Queue
}

// NewConnection - create a new connection to the job queue
func NewConnection(cfg Config) (*Connection, error) {
	dialStr := createConnectionURL(
		cfg.Username, cfg.Password, cfg.Host, cfg.VHost, cfg.Port)
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

	return &Connection{connection, channel, nil}, nil
}

// SetupTopology - declares and configures the RabbitMQ topology
func (c *Connection) SetupTopology() error {
	if err := c.Chan.Qos(1, 0, false); err != nil {
		log.Error.Println("Could not set channel QoS:", err)
		return err
	}

	if err := c.Chan.ExchangeDeclare(
		NewJobExchange, "direct", true, false, false, false, nil); err != nil {
		log.Error.Println("Could not create exchange:", err)
		return err
	}

	q, err := c.Chan.QueueDeclare(NewJobQueue, true, false, false, false, nil)
	if err != nil {
		log.Error.Println("Could not declare job queue:", err)
		return err
	}

	c.Queue = &q

	if err := c.Chan.QueueBind(
		q.Name, RoutingKey, NewJobExchange, false, nil); err != nil {
		log.Error.Println("Could not bind job queue:", err)
		return err
	}

	return nil
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
