package supabaiza_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influx6/sabuhp/supabaiza"
)

func TestMailbox_StartAndStop(t *testing.T) {
	var logger = &LoggerPub{}
	var pubsub = &NoPubSub{}
	var transport = &TransportImpl{
		ConnFunc: func() supabaiza.Conn {
			return nil
		},
		ListenFunc: func(topic string, handler supabaiza.TransportResponse) supabaiza.Channel {
			return nil
		},
		SendToAllFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
		SendToOneFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
	}

	var helloMailbox = supabaiza.NewMailbox(
		context.Background(),
		"hello",
		logger,
		1,
		pubsub,
		transport,
	)

	helloMailbox.Start()

	<-time.After(time.Second)

	helloMailbox.Stop()
}

func TestMailbox_StartAndStopWithCancel(t *testing.T) {
	var logger = &LoggerPub{}
	var pubsub = &NoPubSub{}
	var transport = &TransportImpl{
		ConnFunc: func() supabaiza.Conn {
			return nil
		},
		ListenFunc: func(topic string, handler supabaiza.TransportResponse) supabaiza.Channel {
			return nil
		},
		SendToAllFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
		SendToOneFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var helloMailbox = supabaiza.NewMailbox(
		ctx,
		"hello",
		logger,
		1,
		pubsub,
		transport,
	)

	helloMailbox.Start()

	go func() {
		<-time.After(time.Second)
		canceler()
	}()

	helloMailbox.Wait()
}

func TestMailbox_MessageDelivery(t *testing.T) {
	var logger = &LoggerPub{}
	var pubsub = &NoPubSub{}
	var transport = &TransportImpl{
		ConnFunc: func() supabaiza.Conn {
			return nil
		},
		ListenFunc: func(topic string, handler supabaiza.TransportResponse) supabaiza.Channel {
			return nil
		},
		SendToAllFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
		SendToOneFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var helloMailbox = supabaiza.NewMailbox(
		ctx,
		"hello",
		logger,
		1,
		pubsub,
		transport,
	)

	helloMailbox.Start()

	var message = &supabaiza.Message{
		Topic:    "hello",
		FromAddr: "yay",
		Payload:  supabaiza.BinaryPayload("alex"),
		Metadata: nil,
	}

	var delivered = make(chan struct{})
	var channel = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})

	require.NotNil(t, channel)
	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered

	canceler()
}

func TestMailbox_2Subscribers(t *testing.T) {
	var logger = &LoggerPub{}
	var pubsub = &NoPubSub{}
	var transport = &TransportImpl{
		ConnFunc: func() supabaiza.Conn {
			return nil
		},
		ListenFunc: func(topic string, handler supabaiza.TransportResponse) supabaiza.Channel {
			return nil
		},
		SendToAllFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
		SendToOneFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var helloMailbox = supabaiza.NewMailbox(
		ctx,
		"hello",
		logger,
		1,
		pubsub,
		transport,
	)

	helloMailbox.Start()

	var message = &supabaiza.Message{
		Topic:    "hello",
		FromAddr: "yay",
		Payload:  supabaiza.BinaryPayload("alex"),
		Metadata: nil,
	}

	var delivered = make(chan struct{}, 2)
	var channel1 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel1)

	var channel2 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel2)

	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered
	<-delivered

	canceler()
}

func TestMailbox_3Subscribers_Channel3_Unsubscribed(t *testing.T) {
	var logger = &LoggerPub{}
	var pubsub = &NoPubSub{}
	var transport = &TransportImpl{
		ConnFunc: func() supabaiza.Conn {
			return nil
		},
		ListenFunc: func(topic string, handler supabaiza.TransportResponse) supabaiza.Channel {
			return nil
		},
		SendToAllFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
		SendToOneFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var helloMailbox = supabaiza.NewMailbox(
		ctx,
		"hello",
		logger,
		1,
		pubsub,
		transport,
	)

	helloMailbox.Start()

	var message = &supabaiza.Message{
		Topic:    "hello",
		FromAddr: "yay",
		Payload:  supabaiza.BinaryPayload("alex"),
		Metadata: nil,
	}

	var delivered = make(chan struct{}, 3)

	var channel1 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel1)

	var channel2 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel2)

	var channel3 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel3)

	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered
	<-delivered
	<-delivered

	// close channel 3
	channel3.Close()

	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered
	<-delivered

	require.Len(t, delivered, 0)

	canceler()
}

func TestMailbox_3Subscribers_Channel2_Unsubscribed(t *testing.T) {
	var logger = &LoggerPub{}
	var pubsub = &NoPubSub{}
	var transport = &TransportImpl{
		ConnFunc: func() supabaiza.Conn {
			return nil
		},
		ListenFunc: func(topic string, handler supabaiza.TransportResponse) supabaiza.Channel {
			return nil
		},
		SendToAllFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
		SendToOneFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var helloMailbox = supabaiza.NewMailbox(
		ctx,
		"hello",
		logger,
		1,
		pubsub,
		transport,
	)

	helloMailbox.Start()

	var message = &supabaiza.Message{
		Topic:    "hello",
		FromAddr: "yay",
		Payload:  supabaiza.BinaryPayload("alex"),
		Metadata: nil,
	}

	var delivered = make(chan struct{}, 3)

	var channel1 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel1)

	var channel2 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel2)

	var channel3 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel3)

	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered
	<-delivered
	<-delivered

	// close channel 2
	channel2.Close()

	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered
	<-delivered

	require.Len(t, delivered, 0)

	canceler()
}

func TestMailbox_3Subscribers_Channel1_Unsubscribed(t *testing.T) {
	var logger = &LoggerPub{}
	var pubsub = &NoPubSub{}
	var transport = &TransportImpl{
		ConnFunc: func() supabaiza.Conn {
			return nil
		},
		ListenFunc: func(topic string, handler supabaiza.TransportResponse) supabaiza.Channel {
			return nil
		},
		SendToAllFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
		SendToOneFunc: func(data *supabaiza.Message, timeout time.Duration) error {
			return nil
		},
	}

	var ctx, canceler = context.WithCancel(context.Background())
	var helloMailbox = supabaiza.NewMailbox(
		ctx,
		"hello",
		logger,
		1,
		pubsub,
		transport,
	)

	helloMailbox.Start()

	var message = &supabaiza.Message{
		Topic:    "hello",
		FromAddr: "yay",
		Payload:  supabaiza.BinaryPayload("alex"),
		Metadata: nil,
	}

	var delivered = make(chan struct{}, 3)

	var channel1 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel1)

	var channel2 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel2)

	var channel3 = helloMailbox.Add(func(data *supabaiza.Message, sub supabaiza.PubSub) {
		require.Equal(t, message, data)
		require.NotNil(t, sub)
		delivered <- struct{}{}
	})
	require.NotNil(t, channel3)

	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered
	<-delivered
	<-delivered

	// close channel 1
	channel1.Close()

	require.NoError(t, helloMailbox.Deliver(message))

	<-delivered
	<-delivered

	require.Len(t, delivered, 0)

	canceler()
}
