package hdhomerun

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"
)

type testConnection struct {
	*IOConnection
}

func newTestConnection() *testConnection {
	return &testConnection{
		IOConnection: NewIOConnection(&bytes.Buffer{}),
	}
}

func newTestDevice() *Device {
	return &Device{
		id:         []byte{0x01, 0x02, 0x03, 0x04},
		Connection: newTestConnection(),
	}

}

func TestDefaultAddr(t *testing.T) {
	d := newTestDevice()
	if d.Addr() != nil {
		t.Errorf("Expected nil addr but got %v", d.Addr())
	}
}

func TestGetSet(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		reply         *Packet
		expectedValue TagValue
		expectedErr   reflect.Type
	}{
		{
			name:          "help",
			reply:         getRpy.p,
			expectedValue: getRpy.p.Tags[TagGetSetValue].Value,
		}, {
			name:          "/tuner0/channel",
			value:         "auto:849000000",
			reply:         setRpy.p,
			expectedValue: setRpy.p.Tags[TagGetSetValue].Value,
		}, {
			name:          "help",
			reply:         discoverRpy.p,
			expectedValue: setRpy.p.Tags[TagGetSetValue].Value,
			expectedErr:   reflect.TypeOf(ErrWrongPacketType("")),
		}, {
			name:        "help",
			reply:       getRpyErr.p,
			expectedErr: reflect.TypeOf(ErrRemoteError("")),
		},
	}

	for _, test := range tests {
		d := newTestDevice()
		d.Send(test.reply)

		var value TagValue
		var err error
		if test.value == "" {
			value, err = d.Get(test.name)
		} else {
			value, err = d.Set(test.name, test.value)
		}

		if reflect.TypeOf(err) != test.expectedErr {
			t.Errorf("Expected error %v but got %v", test.expectedErr, reflect.TypeOf(err))
		}

		if err != nil {
			continue
		}

		if !reflect.DeepEqual(value, test.expectedValue) {
			t.Errorf("Expected return value of %s but got %s", test.expectedValue, value)
		}

		err = d.Close()
		if err != nil {
			t.Errorf("Expected no error but got %v", err)
		}
	}
}

func TestDiscover(t *testing.T) {
	tests := []struct {
		reply   testPacket
		devices []string
		err     reflect.Type
	}{
		{
			reply:   discoverRpy,
			devices: []string{hex.EncodeToString(discoverRpy.p.Tags[TagDeviceId].Value)},
		}, {
			reply:   discoverReq,
			devices: []string{},
		}, {
			reply: testPacket{
				p: nil,
				b: []byte{
					0x00,
				},
			},
			devices: []string{},
			err:     reflect.TypeOf(fmt.Errorf("")),
		},
	}

	for _, test := range tests {
		listener, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 65001})
		go func() {
			listener.SetReadDeadline(time.Now().Add(time.Second))
			_, addr, _ := listener.ReadFromUDP(make([]byte, 1024))
			listener.WriteTo(test.reply.b, addr)
			listener.Close()
		}()

		devices := make([]string, 0)
		for result := range Discover(net.IP{127, 0, 0, 1}, time.Second) {
			if reflect.TypeOf(result.Err) != test.err {
				t.Errorf("Expected error type %v but got %v(%v)", test.err, reflect.TypeOf(result.Err), result.Err)
			}

			if result.Err != nil {
				continue
			}
			devices = append(devices, result.Device.ID())

			if result.Device.Addr().String() != "127.0.0.1:65001" {
				t.Errorf("Expected adress 127.0.0.1:65001 but got %s", result.Device.Addr().String())
			}
		}

		if !reflect.DeepEqual(devices, test.devices) {
			t.Errorf("Expected devices %v but got %v", test.devices, devices)
		}
	}
}
