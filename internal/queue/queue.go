package queue

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

const (
	// NewJobExchange - name of RabbitMQ exchange for new jobs
	NewJobExchange string = "jobs.new"
	// NewJobQueue - name of the RabbitMQ queue for new jobs
	NewJobQueue string = "jobs.new"
	// ConsumerName - name to identify the consumer
	ConsumerName string = "cvmfs_job"

	// CompletedJobExchange - name of the RabbitMQ exchange for finished jobs
	CompletedJobExchange string = "jobs.done"
	// SuccessKey - routing/binding key for successful jobs
	SuccessKey string = "success"
	// FailedKey - routing/binding key for failed jobs
	FailedKey string = "failure"
)

// The type of connection to be established to RabbitMQ
const (
	ConsumerConnection = iota
	PublisherConnection
)

// Config - configuration of the job queue
type Config struct {
	Username string
	Password string
	Host     string
	VHost    string
	Port     int
}

// ReadConfig - populate the Config object using the global viper object
//              and the config file
func ReadConfig() (*Config, error) {
	v := viper.Sub("rabbitmq")
	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.vhost", "/cvmfs")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not read RabbitMQ configuration")
	}

	return &cfg, nil
}

// Connection - encapsulates the AMQP connection and channel
type Connection struct {
	Conn              *amqp.Connection
	Chan              *amqp.Channel
	NewJobQueue       *amqp.Queue
	CompletedJobQueue *amqp.Queue
}

// NewConnection - create a new connection to the job queue. connType can
//                 either be ConsumerConnection or PublisherConnection
func NewConnection(cfg *Config, connType int) (*Connection, error) {
	dialStr := createConnectionURL(
		cfg.Username, cfg.Password, cfg.Host, cfg.VHost, cfg.Port)
	connection, err := amqp.Dial(dialStr)
	if err != nil {
		return nil, errors.Wrap(err, "could not open AMQP connection")
	}

	channel, err := connection.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "could not open AMQP channel")
	}

	if err := channel.Qos(1, 0, false); err != nil {
		return nil, errors.Wrap(err, "could not set channel QoS")
	}

	// The exchange for publishing new jobs (to be processed) is durable and
	// non auto-deleted
	if err := channel.ExchangeDeclare(
		NewJobExchange, "direct", true, false, false, false, nil); err != nil {
		return nil, errors.Wrap(err, "could not declare exchange")
	}

	// The exchange for publishing completedd job notifications is not-durable and
	// non auto-deleted
	if err := channel.ExchangeDeclare(
		CompletedJobExchange, "topic", false, false, false, false, nil); err != nil {
		return nil, errors.Wrap(err, "could not declare exchange")
	}

	c := &Connection{connection, channel, nil, nil}

	// In a consumer connection relevant queues are declared and bound
	if connType == ConsumerConnection {
		// Declare and bind a queue for new job notifications (round-robin)
		// This queue is durable, non auto-deleted, non exclusive
		q1, err := channel.QueueDeclare(NewJobQueue, true, false, false, false, nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not declare new job queue")
		}

		if err := channel.QueueBind(
			q1.Name, "", NewJobExchange, false, nil); err != nil {
			return nil, errors.Wrap(err, "could not bind new job queue")
		}

		c.NewJobQueue = &q1

		// Declare and bind a queue for finished job notifications (one-to-all)
		// This queue has an automatically generated name and is exclusive to
		// a single consumer. It is not durable and is auto-deleted
		q2, err := channel.QueueDeclare("", false, true, true, false, nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not declare completed job queue")
		}

		if err := channel.QueueBind(
			q2.Name, "#", CompletedJobExchange, false, nil); err != nil {
			return nil, errors.Wrap(err, "could not bind completed job queue")
		}

		c.CompletedJobQueue = &q2
	}

	return c, nil
}

// Close - closes an established connection to the job queue
func (c *Connection) Close() error {
	return c.Conn.Close()
}

// Publish - publish data (as JSON) to an exchange using the given routing key
func (c *Connection) Publish(exchange string, key string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "could not marshal job into JSON")
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/json",
		Body:         []byte(body),
	}

	if err := c.Chan.Publish(
		exchange, key, true, false, msg); err != nil {
		return errors.Wrap(err, "RabbitMQ publishing failed")
	}

	return nil
}

func createConnectionURL(username string,
	password string, host string, vhost string, port int) string {

	return "amqp://" + username +
		":" + password +
		"@" + host +
		":" + strconv.Itoa(port) + "/" + vhost
}
