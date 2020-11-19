package ssepub

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/influx6/npkg/njson"

	"github.com/influx6/npkg/nxid"

	"github.com/influx6/sabuhp"

	"github.com/influx6/npkg/nerror"
	"github.com/influx6/sabuhp/utils"
)

var (
	newLine         = "\n"
	spaceBytes      = []byte(" ")
	dataHeaderBytes = []byte("data:")
)

type MessageHandler func(message *sabuhp.Message, socket *SSEClient) error

type SSEHub struct {
	maxRetries int
	retryFunc  sabuhp.RetryFunc
	ctx        context.Context
	codec      sabuhp.Codec
	client     *http.Client
	logging    sabuhp.Logger
}

func NewSSEHub(
	ctx context.Context,
	maxRetries int,
	client *http.Client,
	logging sabuhp.Logger,
	codec sabuhp.Codec,
	retryFn sabuhp.RetryFunc,
) *SSEHub {
	if client.CheckRedirect == nil {
		client.CheckRedirect = utils.CheckRedirect
	}
	return &SSEHub{ctx: ctx, maxRetries: maxRetries, client: client, codec: codec, retryFunc: retryFn, logging: logging}
}

func (se *SSEHub) Delete(
	handler MessageHandler,
	route string,
	lastEventIds ...string,
) (*SSEClient, error) {
	return se.For(handler, "Delete", route, nil, lastEventIds...)
}

func (se *SSEHub) Patch(
	handler MessageHandler,
	route string,
	body io.Reader,
	lastEventIds ...string,
) (*SSEClient, error) {
	return se.For(handler, "PATCH", route, body, lastEventIds...)
}

func (se *SSEHub) Post(
	handler MessageHandler,
	route string,
	body io.Reader,
	lastEventIds ...string,
) (*SSEClient, error) {
	return se.For(handler, "POST", route, body, lastEventIds...)
}

func (se *SSEHub) Put(
	handler MessageHandler,
	route string,
	body io.Reader,
	lastEventIds ...string,
) (*SSEClient, error) {
	return se.For(handler, "PUT", route, body, lastEventIds...)
}

func (se *SSEHub) Get(handler MessageHandler, route string, lastEventIds ...string) (*SSEClient, error) {
	return se.For(handler, "GET", route, nil, lastEventIds...)
}

func (se *SSEHub) For(
	handler MessageHandler,
	method string,
	route string,
	body io.Reader,
	lastEventIds ...string,
) (*SSEClient, error) {
	var header = http.Header{}
	header.Set("Cache-Control", "no-cache")
	header.Set("Accept", "text/event-stream")
	if len(lastEventIds) > 0 {
		header.Set(LastEventIdListHeader, strings.Join(lastEventIds, ";"))
	}

	var req, response, err = utils.DoRequest(se.ctx, se.client, method, route, body, header)
	if err != nil {
		return nil, nerror.WrapOnly(err)
	}

	return NewSSEClient(se.maxRetries, handler, req, response, se.codec, se.retryFunc, se.logging, se.client), nil
}

type SSEClient struct {
	maxRetries int
	logger     sabuhp.Logger
	retryFunc  sabuhp.RetryFunc
	handler    MessageHandler
	codec      sabuhp.Codec
	ctx        context.Context
	canceler   context.CancelFunc
	client     *http.Client
	request    *http.Request
	response   *http.Response
	lastId     nxid.ID
	retry      time.Duration
	waiter     sync.WaitGroup
}

func NewSSEClient(
	maxRetries int,
	handler MessageHandler,
	req *http.Request,
	res *http.Response,
	codec sabuhp.Codec,
	retryFn sabuhp.RetryFunc,
	logger sabuhp.Logger,
	reqClient *http.Client,
) *SSEClient {
	if req.Context() == nil {
		panic("Request is required to have a context.Context attached")
	}

	var newCtx, canceler = context.WithCancel(req.Context())
	var client = &SSEClient{
		maxRetries: maxRetries,
		logger:     logger,
		client:     reqClient,
		retryFunc:  retryFn,
		handler:    handler,
		codec:      codec,
		canceler:   canceler,
		ctx:        newCtx,
		request:    req,
		response:   res,
		retry:      0,
	}

	client.waiter.Add(1)
	go client.run()
	return client
}

// Wait blocks till client and it's managing goroutine closes.
func (sc *SSEClient) Wait() {
	sc.waiter.Wait()
}

// Close closes client's request and response cycle
// and waits till managing goroutine is closed.
func (sc *SSEClient) Close() error {
	sc.canceler()
	sc.waiter.Wait()
	return nil
}

func (sc *SSEClient) run() {
	var normalized = utils.NewNormalisedReader(sc.response.Body)
	var reader = bufio.NewReader(normalized)
	var closedOps = false

	var decoding = false
	var data bytes.Buffer
doLoop:
	for {
		select {
		case <-sc.ctx.Done():
			closedOps = true
			break doLoop
		default:
			// do nothing.
		}

		var line, lineErr = reader.ReadString('\n')
		if lineErr != nil {
			njson.Log(sc.logger).New().
				Error().
				Message("failed to read more data").
				String("error", nerror.WrapOnly(lineErr).Error()).
				End()
			break doLoop
		}

		// if we see only a new line then this is the end of
		// an event data section.
		if line == "\n" && decoding {
			decoding = false

			// if we have data, then decode and
			// deliver to handler.
			if data.Len() != 0 {
				njson.Log(sc.logger).New().
					Info().
					Message("received complete data").
					String("data", data.String()).
					End()

				var dataLine = bytes.TrimPrefix(data.Bytes(), dataHeaderBytes)
				dataLine = bytes.TrimPrefix(dataLine, spaceBytes)
				var decodedMessage, decodeErr = sc.codec.Decode(dataLine)
				if decodeErr != nil {
					njson.Log(sc.logger).New().
						Error().
						Message("failed to decode message").
						String("error", nerror.WrapOnly(decodeErr).Error()).
						End()
					break doLoop
				}
				if handleErr := sc.handler(decodedMessage, sc); handleErr != nil {
					njson.Log(sc.logger).New().
						Error().
						Message("failed to handle message").
						String("error", nerror.WrapOnly(handleErr).Error()).
						End()
				}
			}

			continue doLoop
		}

		if line == "\n" && !decoding {
			continue doLoop
		}

		var stripLine = strings.TrimSpace(line)
		if stripLine == SSEStreamHeader {
			decoding = true
			data.Reset()
			continue
		}

		line = strings.TrimSuffix(line, newLine)
		line = strings.TrimPrefix(line, newLine)
		data.WriteString(line)
	}

	if closedOps {
		sc.waiter.Done()
		_ = sc.response.Body.Close()
		return
	}

	sc.reconnect()
}

func (sc *SSEClient) reconnect() {
	var header = http.Header{}
	header.Set("Cache-Control", "no-cache")
	header.Set("Accept", "text/event-stream")
	if !sc.lastId.IsNil() {
		header.Set(LastEventIdListHeader, sc.lastId.String())
	}

	var lastDuration time.Duration
	var retryCount int
	for {
		lastDuration = sc.retryFunc(lastDuration)
		<-time.After(lastDuration)

		var req, response, err = utils.DoRequest(
			sc.ctx,
			sc.client,
			sc.request.Method,
			sc.request.URL.String(),
			nil,
			header,
		)
		if err != nil && retryCount < sc.maxRetries {
			retryCount++
			continue
		}
		if err != nil && retryCount >= sc.maxRetries {
			njson.Log(sc.logger).New().
				Error().
				Message("failed to create request").
				String("error", nerror.WrapOnly(err).Error()).
				End()
			return
		}

		sc.request = req
		sc.response = response
		go sc.run()
		return
	}

}
