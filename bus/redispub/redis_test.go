package redispub

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ewe-studios/sabuhp"
	"github.com/ewe-studios/sabuhp/codecs"
	redis "github.com/go-redis/redis/v8"

	"github.com/stretchr/testify/require"

	"github.com/ewe-studios/sabuhp/testingutils"
)

var codec = &codecs.MessageJsonCodec{}

func TestRedis__Start_Stop_WithCancel(t *testing.T) {
	var ctx, canceler = context.WithCancel(context.Background())

	var logger = &testingutils.LoggerPub{}
	var config Config
	config.Ctx = ctx
	config.Codec = codec
	config.Logger = logger
	config.Redis = redis.Options{
		Network: "tcp",
	}

	var pb, err = PubSub(config)
	require.NoError(t, err)
	require.NotNil(t, pb)

	pb.Start()

	go func() {
		<-time.After(time.Second * 1)
		canceler()
	}()

	pb.Wait()
}

func TestRedis_Start_Stop(t *testing.T) {
	var ctx, canceler = context.WithCancel(context.Background())
	defer canceler()

	var logger = &testingutils.LoggerPub{}
	var config Config
	config.Ctx = ctx
	config.Codec = codec
	config.Logger = logger
	config.Redis = redis.Options{
		Network: "tcp",
	}

	var pb, err = Stream(config)
	require.NoError(t, err)
	require.NotNil(t, pb)

	pb.Start()

	go func() {
		<-time.After(time.Second * 1)
		pb.Stop()
	}()

	pb.Wait()
}

func TestRedis_Stream(t *testing.T) {
	var ctx, canceler = context.WithCancel(context.Background())
	defer canceler()

	var logger = &testingutils.LoggerPub{}
	var config Config
	config.Ctx = ctx
	config.Codec = codec
	config.Logger = logger
	config.Redis = redis.Options{
		Network: "tcp",
	}

	var pb, err = Stream(config)
	require.NoError(t, err)
	require.NotNil(t, pb)

	pb.Start()

	var content = []byte("\"yes\"")
	var whyMessage = sabuhp.NewMessage(sabuhp.T("why"), "me", content)
	var whatMessage = sabuhp.NewMessage(sabuhp.T("what"), "me", content)

	var delivered sync.WaitGroup
	delivered.Add(2)

	var channel = pb.Listen(
		"what",
		"*",
		sabuhp.TransportResponseFunc(
			func(ctx context.Context, message sabuhp.Message, transport sabuhp.Transport) sabuhp.MessageErr {
				delivered.Done()
				transport.Bus.Send(whyMessage)
				return nil
			}))

	require.NoError(t, channel.Err())

	defer channel.Close()

	var channel2 = pb.Listen("why", "*", sabuhp.TransportResponseFunc(
		func(ctx context.Context, message sabuhp.Message, transport sabuhp.Transport) sabuhp.MessageErr {
			delivered.Done()
			return nil
		}))

	require.NoError(t, channel2.Err())

	defer channel2.Close()

	pb.Send(whatMessage)

	delivered.Wait()

	canceler()
	pb.Wait()
}

func TestRedis_PubSub(t *testing.T) {
	var ctx, canceler = context.WithCancel(context.Background())
	defer canceler()

	var logger = &testingutils.LoggerPub{}
	var config Config
	config.Ctx = ctx
	config.Codec = codec
	config.Logger = logger
	config.Redis = redis.Options{
		Network: "tcp",
	}

	var pb, err = PubSub(config)
	require.NoError(t, err)
	require.NotNil(t, pb)

	pb.Start()

	var content = []byte("\"yes\"")
	var whyMessage = sabuhp.NewMessage(sabuhp.T("why"), "me", content)
	var whatMessage = sabuhp.NewMessage(sabuhp.T("what"), "me", content)

	var delivered sync.WaitGroup
	delivered.Add(2)

	var channel = pb.Listen(
		"what",
		"*",
		sabuhp.TransportResponseFunc(
			func(ctx context.Context, message sabuhp.Message, transport sabuhp.Transport) sabuhp.MessageErr {
				delivered.Done()
				transport.Bus.Send(whyMessage)
				return nil
			}))

	require.NoError(t, channel.Err())

	defer channel.Close()

	var channel2 = pb.Listen("why", "*", sabuhp.TransportResponseFunc(
		func(ctx context.Context, message sabuhp.Message, transport sabuhp.Transport) sabuhp.MessageErr {
			delivered.Done()
			return nil
		}))

	require.NoError(t, channel2.Err())

	defer channel2.Close()

	pb.Send(whatMessage)

	delivered.Wait()

	canceler()
	pb.Wait()
}

func TestRedis_PubSub_WithReply(t *testing.T) {
	var ctx, canceler = context.WithCancel(context.Background())
	defer canceler()

	var logger = &testingutils.LoggerPub{}
	var config Config
	config.Ctx = ctx
	config.Codec = codec
	config.Logger = logger
	config.Redis = redis.Options{
		Network: "tcp",
	}

	var pb, err = PubSub(config)
	require.NoError(t, err)
	require.NotNil(t, pb)

	pb.Start()

	var content = []byte("\"yes\"")

	var whyMessage = sabuhp.NewMessage(sabuhp.T("why"), "me", content)
	whyMessage.ReplyGroup = "*"

	var whyReplyMessage = sabuhp.NewMessage(whyMessage.Topic.ReplyTopic(), "me", content)
	whyReplyMessage.ReplyGroup = "*"
	whyReplyMessage.Bytes = []byte("Yo!")

	var delivered sync.WaitGroup
	delivered.Add(1)

	var channel = pb.Listen(
		whyMessage.Topic.String(),
		"*",
		sabuhp.TransportResponseFunc(
			func(ctx context.Context, message sabuhp.Message, transport sabuhp.Transport) sabuhp.MessageErr {
				fmt.Printf("Received message: %+s\n", message)
				delivered.Done()
				transport.Bus.Send(whyReplyMessage)
				return nil
			}))

	require.NoError(t, channel.Err())

	defer channel.Close()

	var replyFT = pb.SendForReply(time.Minute, whyMessage.Topic, "*", whyMessage)
	var replyMsg, replyErr = replyFT.Get()
	require.NoError(t, replyErr)
	require.NotNil(t, replyMsg)

	var rm = replyMsg.(sabuhp.Message)

	require.Equal(t, "Yo!", string(rm.Bytes))

	delivered.Wait()

	canceler()
	pb.Wait()
}
