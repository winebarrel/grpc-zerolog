package grpc_zerolog_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	grpc_zerolog "github.com/winebarrel/grpc-zerolog"
)

func TestDurationToTimeMillisField(t *testing.T) {
	_, val := grpc_zerolog.DurationToTimeMillisField(time.Microsecond * 100)
	assert.Equal(t, val.(float32), float32(0.1), "sub millisecond values should be correct")
}
