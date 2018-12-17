package cvmfs

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	// newJobExchange - name of RabbitMQ exchange for new jobs
	newJobExchange string = "jobs.new"
	// newJobQueue - name of the RabbitMQ queue for new jobs
	newJobQueue string = "jobs.new"

	// completedJobExchange - name of the RabbitMQ exchange for finished jobs
	completedJobExchange string = "jobs.done"
	// successKey - routing/binding key for successful jobs
	successKey string = "success"
	// failedKey - routing/binding key for failed jobs
	failedKey string = "failure"
)

// The type of connection to be established to RabbitMQ
const (
	consumerConnection = iota
	publisherConnection
)

// QueueClient - encapsulates the AMQP connection and channel
type QueueClient struct {
	Conn              *amqp.Connection
	Chan              *amqp.Channel
	NewJobQueue       *amqp.Queue
	CompletedJobQueue *amqp.Queue
}

// NewQueueClient - create a new connection to the job queue. connType can
//             either be ConsumerConnection or publisherConnection
func NewQueueClient(cfg *QueueConfig, connType int) (*QueueClient, error) {
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
		newJobExchange, "direct", true, false, false, false, nil); err != nil {
		return nil, errors.Wrap(err, "could not declare exchange")
	}

	// The exchange for publishing completedd job notifications is not-durable and
	// non auto-deleted
	if err := channel.ExchangeDeclare(
		completedJobExchange, "topic", false, false, false, false, nil); err != nil {
		return nil, errors.Wrap(err, "could not declare exchange")
	}

	c := &QueueClient{connection, channel, nil, nil}

	// In a consumer connection relevant queues are declared and bound
	if connType == consumerConnection {
		// Declare and bind a queue for new job notifications (round-robin)
		// This queue is durable, non auto-deleted, non exclusive
		q1, err := channel.QueueDeclare(newJobQueue, true, false, false, false, nil)
		if err != nil {
			return nil, errors.Wrap(err, "could not declare new job queue")
		}

		if err := channel.QueueBind(
			q1.Name, "", newJobExchange, false, nil); err != nil {
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
			q2.Name, "#", completedJobExchange, false, nil); err != nil {
			return nil, errors.Wrap(err, "could not bind completed job queue")
		}

		c.CompletedJobQueue = &q2

		go func() {
			ch := c.Chan.NotifyClose(make(chan *amqp.Error))
			err, ok := <-ch
			if ok {
				LogError.Println(
					errors.Wrap(err, "connection to job queue closed"))
				os.Exit(1)
			}
		}()
	}

	return c, nil
}

// Close - closes an established connection to the job queue
func (c *QueueClient) Close() error {
	return c.Conn.Close()
}

// publish - publish data (as JSON) to an exchange using the given routing key
func (c *QueueClient) publish(exchange string, key string, data interface{}) error {
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
