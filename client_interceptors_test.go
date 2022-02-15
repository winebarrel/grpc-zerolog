package grpc_zerolog_test

import (
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	grpc_zerolog "github.com/winebarrel/grpc-zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
)

func customClientCodeToLevel(c codes.Code) zerolog.Level {
	if c == codes.Unauthenticated {
		// Make this a special case for tests, and an error.
		return zerolog.ErrorLevel
	}
	level := grpc_zerolog.DefaultClientCodeToLevel(c)
	return level
}

func TestZerologClientSuite(t *testing.T) {
	opts := []grpc_zerolog.Option{
		grpc_zerolog.WithLevels(customClientCodeToLevel),
	}
	b := newZerologBaseSuite(t)
	b.InterceptorTestSuite.ClientOpts = []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_zerolog.UnaryClientInterceptor(b.logger, opts...)),
		grpc.WithStreamInterceptor(grpc_zerolog.StreamClientInterceptor(b.logger, opts...)),
	}
	suite.Run(t, &zerologClientSuite{b})
}

type zerologClientSuite struct {
	*zerologBaseSuite
}

func (s *zerologClientSuite) TestPing() {
	_, err := s.Client.Ping(s.SimpleCtx(), goodPing)
	assert.NoError(s.T(), err, "there must be not be an on a successful call")

	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	assert.Equal(s.T(), msgs[0]["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain the correct service name")
	assert.Equal(s.T(), msgs[0]["grpc.method"], "Ping", "all lines must contain the correct method name")
	assert.Equal(s.T(), msgs[0]["message"], "finished client unary call", "handler's message must contain the correct message")
	assert.Equal(s.T(), msgs[0]["span.kind"], "client", "all lines must contain the kind of call (client)")
	assert.Equal(s.T(), msgs[0]["level"], "debug", "OK codes must be logged on debug level.")

	assert.Contains(s.T(), msgs[0], "grpc.time_ms", "interceptor log statement should contain execution time (duration in ms)")
}

func (s *zerologClientSuite) TestPingList() {
	stream, err := s.Client.PingList(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "should not fail on establishing the stream")
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(s.T(), err, "reading stream should not fail")
	}

	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	assert.Equal(s.T(), msgs[0]["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain the correct service name")
	assert.Equal(s.T(), msgs[0]["grpc.method"], "PingList", "all lines must contain the correct method name")
	assert.Equal(s.T(), msgs[0]["message"], "finished client streaming call", "handler's message must contain the correct message")
	assert.Equal(s.T(), msgs[0]["span.kind"], "client", "all lines must contain the kind of call (client)")
	assert.Equal(s.T(), msgs[0]["level"], "debug", "OK codes must be logged on debug level.")
	assert.Contains(s.T(), msgs[0], "grpc.time_ms", "interceptor log statement should contain execution time (duration in ms)")
}

func (s *zerologClientSuite) TestPingError_WithCustomLevels() {
	for _, tcase := range []struct {
		code  codes.Code
		level zerolog.Level
		msg   string
	}{
		{
			code:  codes.Internal,
			level: zerolog.WarnLevel,
			msg:   "Internal must remap to ErrorLevel in DefaultClientCodeToLevel",
		},
		{
			code:  codes.NotFound,
			level: zerolog.DebugLevel,
			msg:   "NotFound must remap to InfoLevel in DefaultClientCodeToLevel",
		},
		{
			code:  codes.FailedPrecondition,
			level: zerolog.DebugLevel,
			msg:   "FailedPrecondition must remap to WarnLevel in DefaultClientCodeToLevel",
		},
		{
			code:  codes.Unauthenticated,
			level: zerolog.ErrorLevel,
			msg:   "Unauthenticated is overwritten to ErrorLevel with customClientCodeToLevel override, which probably didn't work",
		},
	} {
		s.SetupTest()
		_, err := s.Client.PingError(
			s.SimpleCtx(),
			&pb_testproto.PingRequest{Value: "something", ErrorCodeReturned: uint32(tcase.code)})

		assert.Error(s.T(), err, "each call here must return an error")

		msgs := s.getOutputJSONs()
		require.Len(s.T(), msgs, 1, "only a single log message is printed")

		assert.Equal(s.T(), msgs[0]["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain the correct service name")
		assert.Equal(s.T(), msgs[0]["grpc.method"], "PingError", "all lines must contain the correct method name")
		assert.Equal(s.T(), msgs[0]["grpc.code"], tcase.code.String(), "all lines must contain a grpc code")
		assert.Equal(s.T(), msgs[0]["level"], tcase.level.String(), tcase.msg)
	}
}

func TestZerologClientOverrideSuite(t *testing.T) {
	opts := []grpc_zerolog.Option{
		grpc_zerolog.WithDurationField(grpc_zerolog.DurationToDurationField),
	}
	b := newZerologBaseSuite(t)
	b.InterceptorTestSuite.ClientOpts = []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_zerolog.UnaryClientInterceptor(b.logger, opts...)),
		grpc.WithStreamInterceptor(grpc_zerolog.StreamClientInterceptor(b.logger, opts...)),
	}
	suite.Run(t, &zerologClientOverrideSuite{b})
}

type zerologClientOverrideSuite struct {
	*zerologBaseSuite
}

func (s *zerologClientOverrideSuite) TestPing_HasOverrides() {
	_, err := s.Client.Ping(s.SimpleCtx(), goodPing)
	assert.NoError(s.T(), err, "there must be not be an on a successful call")

	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")
	assert.Equal(s.T(), msgs[0]["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain the correct service name")
	assert.Equal(s.T(), msgs[0]["grpc.method"], "Ping", "all lines must contain the correct method name")
	assert.Equal(s.T(), msgs[0]["message"], "finished client unary call", "handler's message must contain the correct message")

	assert.NotContains(s.T(), msgs[0], "grpc.time_ms", "message must not contain default duration")
	assert.Contains(s.T(), msgs[0], "grpc.duration", "message must contain overridden duration")
}

func (s *zerologClientOverrideSuite) TestPingList_HasOverrides() {
	stream, err := s.Client.PingList(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "should not fail on establishing the stream")
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(s.T(), err, "reading stream should not fail")
	}

	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	assert.Equal(s.T(), msgs[0]["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain the correct service name")
	assert.Equal(s.T(), msgs[0]["grpc.method"], "PingList", "all lines must contain the correct method name")
	assert.Equal(s.T(), msgs[0]["message"], "finished client streaming call", "log message must be correct")
	assert.Equal(s.T(), msgs[0]["span.kind"], "client", "all lines must contain the kind of call (client)")
	assert.Equal(s.T(), msgs[0]["level"], "debug", "OK codes must be logged on debug level.")

	assert.NotContains(s.T(), msgs[0], "grpc.time_ms", "message must not contain default duration")
	assert.Contains(s.T(), msgs[0], "grpc.duration", "message must contain overridden duration")
}

func TestZerologClientMessageProducerSuite(t *testing.T) {
	opts := []grpc_zerolog.Option{
		grpc_zerolog.WithMessageProducer(StubMessageProducer),
	}
	b := newZerologBaseSuite(t)
	b.InterceptorTestSuite.ClientOpts = []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_zerolog.UnaryClientInterceptor(b.logger, opts...)),
		grpc.WithStreamInterceptor(grpc_zerolog.StreamClientInterceptor(b.logger, opts...)),
	}
	suite.Run(t, &zerologClientMessageProducerSuite{b})
}

type zerologClientMessageProducerSuite struct {
	*zerologBaseSuite
}

func (s *zerologClientMessageProducerSuite) TestPing_HasOverriddenMessageProducer() {
	_, err := s.Client.Ping(s.SimpleCtx(), goodPing)
	assert.NoError(s.T(), err, "there must be not be an on a successful call")

	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	assert.Equal(s.T(), msgs[0]["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain the correct service name")
	assert.Equal(s.T(), msgs[0]["grpc.method"], "Ping", "all lines must contain the correct method name")
	assert.Equal(s.T(), msgs[0]["message"], "custom message", "handler's message must contain the correct message")
	assert.Equal(s.T(), msgs[0]["span.kind"], "client", "all lines must contain the kind of call (client)")
	assert.Equal(s.T(), msgs[0]["level"], "debug", "OK codes must be logged on debug level.")

	assert.Contains(s.T(), msgs[0], "grpc.time_ms", "interceptor log statement should contain execution time (duration in ms)")
}
