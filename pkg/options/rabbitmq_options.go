// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"
	"net"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/pflag"
)

var _ IOptions = (*RabbitMQOptions)(nil)

// RabbitMQOptions defines options for the RabbitMQ message queue.
type RabbitMQOptions struct {
	// URL is the AMQP connection URL.
	URL string `json:"url" mapstructure:"url"`
	// Exchange is the exchange to publish/consume messages.
	Exchange string `json:"exchange" mapstructure:"exchange"`
	// ExchangeType is the exchange type (direct/topic/fanout/headers).
	ExchangeType string `json:"exchange-type" mapstructure:"exchange-type"`
	// Queue is the queue to consume messages.
	Queue string `json:"queue" mapstructure:"queue"`
	// RoutingKey is the routing key used to bind the queue to the exchange.
	RoutingKey string `json:"routing-key" mapstructure:"routing-key"`
	// Timeout is the dial timeout.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewRabbitMQOptions create a `zero` value instance.
func NewRabbitMQOptions() *RabbitMQOptions {
	return &RabbitMQOptions{
		URL:          "amqp://guest:guest@127.0.0.1:5672/",
		ExchangeType: "topic",
		Timeout:      3 * time.Second,
	}
}

// Validate verifies flags passed to RabbitMQOptions.
func (o *RabbitMQOptions) Validate() []error {
	errs := []error{}

	if o.URL == "" {
		errs = append(errs, fmt.Errorf("--rabbitmq.url can not be empty"))
	}

	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("--rabbitmq.timeout cannot be negative"))
	}

	return errs
}

// AddFlags adds flags related to rabbitmq to the specified FlagSet.
func (o *RabbitMQOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.URL, fullPrefix+".url", o.URL, "AMQP connection URL of the rabbitmq server.")
	fs.StringVar(&o.Exchange, fullPrefix+".exchange", o.Exchange, "Exchange to publish/consume messages.")
	fs.StringVar(&o.ExchangeType, fullPrefix+".exchange-type", o.ExchangeType, "Exchange type (direct/topic/fanout/headers).")
	fs.StringVar(&o.Queue, fullPrefix+".queue", o.Queue, "Queue to consume messages.")
	fs.StringVar(&o.RoutingKey, fullPrefix+".routing-key", o.RoutingKey, "Routing key used to bind the queue to the exchange.")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Dial timeout.")
}

// NewConnection creates a new RabbitMQ connection based on the provided options.
func (o *RabbitMQOptions) NewConnection() (*amqp.Connection, error) {
	conn, err := amqp.DialConfig(o.URL, amqp.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, o.Timeout)
		},
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// NewChannel creates a new RabbitMQ channel from a newly-created connection.
// The caller is responsible for closing both the channel and the connection.
func (o *RabbitMQOptions) NewChannel() (*amqp.Channel, error) {
	conn, err := o.NewConnection()
	if err != nil {
		return nil, err
	}
	return conn.Channel()
}
