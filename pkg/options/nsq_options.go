// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/spf13/pflag"
)

var _ IOptions = (*NSQOptions)(nil)

// NSQOptions defines options for the NSQ message queue.
type NSQOptions struct {
	// NSQDAddr is the address of the nsqd TCP server (producer).
	NSQDAddr string `json:"nsqd-addr" mapstructure:"nsqd-addr"`
	// LookupdAddr is the address of the nsqlookupd HTTP server (consumer).
	LookupdAddr string `json:"lookupd-addr" mapstructure:"lookupd-addr"`
	// Topic is the topic to produce/consume messages.
	Topic string `json:"topic" mapstructure:"topic"`
	// Channel is the consumer channel.
	Channel string `json:"channel" mapstructure:"channel"`
	// MaxInFlight is the maximum number of messages to allow in flight.
	MaxInFlight int `json:"max-in-flight" mapstructure:"max-in-flight"`
	// Timeout is the network timeout.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewNSQOptions create a `zero` value instance.
func NewNSQOptions() *NSQOptions {
	return &NSQOptions{
		NSQDAddr:    "127.0.0.1:4150",
		LookupdAddr: "127.0.0.1:4161",
		MaxInFlight: 1,
		Timeout:     3 * time.Second,
	}
}

// Validate verifies flags passed to NSQOptions.
func (o *NSQOptions) Validate() []error {
	errs := []error{}

	if o.NSQDAddr == "" && o.LookupdAddr == "" {
		errs = append(errs, fmt.Errorf("--nsq.nsqd-addr and --nsq.lookupd-addr can not both be empty"))
	}

	if o.Topic == "" {
		errs = append(errs, fmt.Errorf("--nsq.topic can not be empty"))
	}

	return errs
}

// AddFlags adds flags related to nsq to the specified FlagSet.
func (o *NSQOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.NSQDAddr, fullPrefix+".nsqd-addr", o.NSQDAddr, "Address of the nsqd TCP server.")
	fs.StringVar(&o.LookupdAddr, fullPrefix+".lookupd-addr", o.LookupdAddr, "Address of the nsqlookupd HTTP server.")
	fs.StringVar(&o.Topic, fullPrefix+".topic", o.Topic, "Topic to produce/consume messages.")
	fs.StringVar(&o.Channel, fullPrefix+".channel", o.Channel, "Consumer channel.")
	fs.IntVar(&o.MaxInFlight, fullPrefix+".max-in-flight", o.MaxInFlight, "Maximum number of messages in flight.")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Network timeout.")
}

// NewProducer creates a new NSQ producer based on the provided options.
func (o *NSQOptions) NewProducer() (*nsq.Producer, error) {
	config := nsq.NewConfig()
	config.DialTimeout = o.Timeout
	return nsq.NewProducer(o.NSQDAddr, config)
}

// NewConsumer creates a new NSQ consumer based on the provided options.
// It registers the given handler and connects to nsqlookupd (preferred) or nsqd.
func (o *NSQOptions) NewConsumer(handler nsq.Handler) (*nsq.Consumer, error) {
	config := nsq.NewConfig()
	config.DialTimeout = o.Timeout
	config.MaxInFlight = o.MaxInFlight

	consumer, err := nsq.NewConsumer(o.Topic, o.Channel, config)
	if err != nil {
		return nil, err
	}
	consumer.AddHandler(handler)

	if o.LookupdAddr != "" {
		if err := consumer.ConnectToNSQLookupd(o.LookupdAddr); err != nil {
			return nil, err
		}
	} else if err := consumer.ConnectToNSQD(o.NSQDAddr); err != nil {
		return nil, err
	}

	return consumer, nil
}
