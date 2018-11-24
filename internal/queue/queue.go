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
	// RoutingKey - routing key used when publishing jobs
	RoutingKey string = ""
	// ConsumerName - name to identify the consumer
	ConsumerName string = "cvmfs_job"

	// CompletedJobExchange - name of the RabbitMQ exchange for finished jobs
	CompletedJobExchange string = "jobs.done"
	// SuccessKey - routing/binding key for successful jobs
	SuccessKey string = "success"
	// FailedKey - routing/binding key for failed jobs
	FailedKey string = "failure"
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
	Conn  *amqp.Connection
	Chan  *amqp.Channel
	Queue *amqp.Queue
}

// NewConnection - create a new connection to the job queue
func NewConnection(cfg *Config) (*Connection, error) {
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

	if err := channel.ExchangeDeclare(
		NewJobExchange, "direct", true, false, false, false, nil); err != nil {
		return nil, errors.Wrap(err, "could not declare exchange")
	}

	if err := channel.ExchangeDeclare(
		CompletedJobExchange, "topic", true, false, false, false, nil); err != nil {
		return nil, errors.Wrap(err, "could not declare exchange")
	}

	q, err := channel.QueueDeclare(NewJobQueue, true, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not declare job queue")
	}

	if err := channel.QueueBind(
		q.Name, RoutingKey, NewJobExchange, false, nil); err != nil {
		return nil, errors.Wrap(err, "could not bind job queue")
	}

	return &Connection{connection, channel, &q}, nil
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
