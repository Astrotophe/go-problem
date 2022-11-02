/*
* Copyright (c) 2017-2020. Canal+ Group
* All rights reserved
 */
package kafka

import (
	"context"
	"fmt"
	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

var logDataDog *log.Logger

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	config   *sarama.Config
	brokers  []string
	readChan chan *ReadModel
}

func NewConsumer(config *sarama.Config, brokers []string, readChan chan *ReadModel, logg *log.Logger) *Consumer {
    logDataDog = logg
	return &Consumer{
		config:   config,
		readChan: readChan,
		brokers:  brokers,
	}
}

type ReadModel struct {
	Session     sarama.ConsumerGroupSession
	Message     *sarama.ConsumerMessage
}

func (c *Consumer) Consume(ctx context.Context, group string, topics []string, doneChan chan bool) {
	client, err := sarama.NewConsumerGroup(c.brokers, group, c.config)
	if err != nil {
		logDataDog.Error("Error creating consumer group client: %v", err)
	}
	done := false
	for !done {
		select {
		case <-ctx.Done():
			err := client.Close()
			if err != nil {
				logDataDog.Error("ERROR: Failed to close Kafka client:", err)
			}
			done = true
		default:
			if err := client.Consume(ctx, topics, c); err != nil {
				logDataDog.Error("Error from consumer: %v", err)
			}
		}
	}
	doneChan <- true
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}


// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		logDataDog.Info(fmt.Sprintf("_______ New message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic))
		c.readChan <- &ReadModel{session, message}
	}

	return nil
}
