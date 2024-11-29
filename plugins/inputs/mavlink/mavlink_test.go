package mavlink

import (
	"testing"

	"github.com/influxdata/telegraf/internal/choice"
	"github.com/stretchr/testify/require"
)

// Test that a serial port URL can be parsed.
func TestParseSerialFcuUrl(t *testing.T) {
	testConfig := Mavlink{
		FcuURL: "serial://dev/ttyACM0:115200",
	}

	_, err := ParseMavlinkEndpointConfig(&testConfig)
	require.NoError(t, err)
}

// Test that a UDP client URL can be parsed.
func TestParseUDPClientFcuUrl(t *testing.T) {
	testConfig := Mavlink{
		FcuURL: "udp://192.168.1.12:14550",
	}

	_, err := ParseMavlinkEndpointConfig(&testConfig)
	require.NoError(t, err)
}

// Test that a UDP server URL can be parsed.
func TestParseUDPServerFcuUrl(t *testing.T) {
	testConfig := Mavlink{
		FcuURL: "udp://:14540",
	}

	_, err := ParseMavlinkEndpointConfig(&testConfig)
	require.NoError(t, err)
}

// Test that a TCP client URL can be parsed.
func TestParseTCPClientFcuUrl(t *testing.T) {
	testConfig := Mavlink{
		FcuURL: "tcp://192.168.1.12:14550",
	}

	_, err := ParseMavlinkEndpointConfig(&testConfig)
	require.NoError(t, err)
}

// Test that an invalid URL is caught.
func TestParseInvalidFcuUrl(t *testing.T) {
	testConfig := Mavlink{
		FcuURL: "ftp://not-a-valid-fcu-url",
	}

	_, err := ParseMavlinkEndpointConfig(&testConfig)
	require.Equal(t, "mavlink setup error: invalid fcu_url", err.Error())
}

func TestStringContains(t *testing.T) {
	testArr := []string{"test1", "test2", "test3"}
	require.True(t, choice.Contains("test1", testArr))
	require.True(t, choice.Contains("test2", testArr))
	require.True(t, choice.Contains("test3", testArr))
	require.False(t, choice.Contains("test4", testArr))
}

func TestConvertToSnakeCase(t *testing.T) {
	require.Equal(t, "", ConvertToSnakeCase(""))
	require.Equal(t, "camel_case", ConvertToSnakeCase("CamelCase"))
	require.Equal(t, "camel_camel_case", ConvertToSnakeCase("CamelCamelCase"))
	require.Equal(t, "snake_case", ConvertToSnakeCase("snake_case"))
	require.Equal(t, "snake_case", ConvertToSnakeCase("SNAKE_CASE"))
}
