/*
* Copyright (c) 2017-2020. Canal+ Group
* All rights reserved
 */
package kafka

import (
	"context"
	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
	"sync"
)

// consumer represents a Sarama consumer group consumer
type consumer struct {
	config   *sarama.Config
	brokers  []string
	readChan chan *ReadModel
	logger   *log.Logger
}

type Consumer interface {
	sarama.ConsumerGroupHandler
	Consume(ctx context.Context, group string, topics []string, wg *sync.WaitGroup)
}

func NewConsumer(config *sarama.Config, brokers []string, readChan chan *ReadModel, logger *log.Logger) Consumer {
	return &consumer{
		config:   config,
		readChan: readChan,
		brokers:  brokers,
		logger:   logger,
	}
}

type ReadModel struct {
	Session sarama.ConsumerGroupSession
	Message *sarama.ConsumerMessage
}

func (c *consumer) Consume(ctx context.Context, group string, topics []string, wg *sync.WaitGroup) {
	client, err := sarama.NewConsumerGroup(c.brokers, group, c.config)
	if err != nil {
		c.logger.WithError(err).Error("Error creating consumer group client")
	}
	for {
		select {
		case <-ctx.Done():
			err := client.Close()
			if err != nil {
				c.logger.WithError(err).Error("ERROR: Failed to close Kafka client")
			}
			wg.Done()
			return
		default:
			if err := client.Consume(ctx, topics, c); err != nil {
				c.logger.WithError(err).Error("Error from consumer")
			}
		}
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/main/consumer_group.go#L27-L29
	for {
		select {
		case message := <-claim.Messages():
			c.logger.Infof("_______ New message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			c.readChan <- &ReadModel{session, message}

		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/Shopify/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}
