package network

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func TestNetworkFixtures(t *testing.T) {
	ctx := context.Background()
	for _, device := range []string{"rax3000m", "x86_generic"} {
		for _, fixture := range []struct {
			call func(context.Context, *Client) error
			file string
		}{
			{call: checkDeviceFixture, file: "network_device_status.json"},
			{call: checkInterfaceFixture, file: "network_interface_dump.json"},
			{call: checkWirelessFixture, file: "network_wireless_status.json"},
		} {
			raw := readNetworkFixture(t, device, fixture.file)
			client := fixtureNetworkClient(raw)
			err := fixture.call(ctx, client)
			if err != nil {
				t.Errorf("%s/%s: %v", device, fixture.file, err)

				continue
			}
			assertExpandedNetworkFixture(t, device, fixture.file, client)
		}
	}
}

func assertExpandedNetworkFixture(t *testing.T, profile, name string, client *Client) {
	t.Helper()
	switch name {
	case "network_device_status.json":
		assertDeviceFields(t, profile, client)
	case "network_interface_dump.json":
		assertInterfaceFields(t, profile, client)
	case "network_wireless_status.json":
		assertRadioFields(t, profile, client)
	}
}

func assertDeviceFields(t *testing.T, profile string, client *Client) {
	t.Helper()
	devices, err := client.Devices().Status(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	bridge, ok := devices["br-lan"]
	if !ok || len(bridge.BridgeMembers) == 0 {
		t.Fatalf("br-lan bridge members = %+v", bridge.BridgeMembers)
	}
	port, ok := devices["eth0"]
	if !ok || port.FlowControl == nil {
		t.Fatalf("eth0 flow control = %+v", port.FlowControl)
	}
	if profile == "rax3000m" {
		if len(port.FlowControl.Negotiated) == 0 || bridge.Statistics.TxDropped != 2 ||
			bridge.Statistics.Multicast != 3747 || bridge.Statistics.Collisions != 0 {
			t.Fatalf("device expanded counters = %+v, flow control = %+v", bridge.Statistics, port.FlowControl)
		}
		if devices["lan1"].Conduit != "eth0" {
			t.Fatalf("lan1 conduit = %q", devices["lan1"].Conduit)
		}
	}
}

func assertInterfaceFields(t *testing.T, profile string, client *Client) {
	t.Helper()
	interfaces, err := client.DumpInterfaces(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if profile == "x86_generic" {
		for _, iface := range interfaces {
			if iface.Interface == "wan" {
				if iface.Data["dhcpserver"] != "172.17.64.1" || iface.Data["leasetime"] != float64(86400) {
					t.Fatalf("wan data = %#v", iface.Data)
				}

				return
			}
		}
		t.Fatal("wan interface not found")
	}
	for _, iface := range interfaces {
		if iface.Interface == "wg0" {
			if len(iface.Errors) == 0 || iface.Errors[0].Subsystem != "interface" || iface.Errors[0].Code != "NO_DEVICE" {
				t.Fatalf("wg0 errors = %+v", iface.Errors)
			}

			return
		}
	}
	t.Fatal("wg0 interface not found")
}

func assertRadioFields(t *testing.T, profile string, client *Client) {
	t.Helper()
	if profile != "rax3000m" {
		return
	}
	radios, err := client.Wireless().Status(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	radio := radios["radio0"]
	if radio.Config["band"] != "2g" || len(radio.Interfaces) == 0 || radio.Interfaces[0].Config["mode"] != "ap" {
		t.Fatalf("radio0 expanded config = %+v", radio)
	}
}

func checkDeviceFixture(ctx context.Context, client *Client) error {
	_, err := client.Devices().Status(ctx, "")

	return err
}

func checkInterfaceFixture(ctx context.Context, client *Client) error {
	_, err := client.DumpInterfaces(ctx)

	return err
}

func checkWirelessFixture(ctx context.Context, client *Client) error {
	_, err := client.Wireless().Status(ctx, "")

	return err
}

func fixtureNetworkClient(raw []byte) *Client {
	caller := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return goubus.NewJSONResult(raw), nil
	})

	return New(caller)
}

func readNetworkFixture(t *testing.T, device, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "testdata", device, name)
	// #nosec G304 -- the fixture path is limited to the repository's checked-in testdata.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return data
}

func TestWirelessValidateAndNilCaller(t *testing.T) {
	var gotMethod string
	client := New(goubus.CallerFunc(func(_ context.Context, _, method string, _ any) (goubus.Result, error) {
		gotMethod = method

		return goubus.NewJSONResult([]byte(`{"valid":true}`)), nil
	}))
	_, err := client.Wireless().Validate(context.Background(), "radio0")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != "get_validate" {
		t.Fatalf("method=%q, want get_validate", gotMethod)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("New(nil) did not panic")
		}
	}()
	New(nil)
}
