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
			}
		}
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
