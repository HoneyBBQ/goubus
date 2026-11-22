package luci

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

var errTemporaryLocaltimeFailure = errors.New("temporary localtime failure")

type fixtureCheck func(*Client, context.Context) error

func TestLuCIFixtures(t *testing.T) {
	ctx := context.Background()
	checks := luciFixtureChecks()
	for _, device := range []string{"rax3000m", "x86_generic"} {
		pattern := filepath.Join("..", "..", "internal", "testdata", device, "luci*.json")
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) == 0 {
			t.Fatalf("no LuCI fixtures matched %s", pattern)
		}
		for _, file := range files {
			// #nosec G304 -- fixture paths are generated from fixed device names and a fixed glob.
			raw, readErr := os.ReadFile(file)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(raw) == 0 {
				continue
			}
			if !json.Valid(raw) {
				t.Fatalf("invalid JSON in %s", file)
			}
			check := checks[filepath.Base(file)]
			if check == nil {
				continue
			}
			method := "getUnixtime"
			if filepath.Base(file) == "luci_getLocaltime.json" {
				method = "getLocaltime"
			}
			client := fixtureLuciClient(raw, WithTimeMethod(method))
			err := check(client, ctx)
			if err != nil {
				t.Errorf("%s/%s: %v", device, filepath.Base(file), err)
			}
		}
	}
}

func luciFixtureChecks() map[string]fixtureCheck {
	return map[string]fixtureCheck{
		"luci_getBlockDevices.json": fixtureCheckValue((*Client).BlockDevices),
		"luci_getFeatures.json":     fixtureCheckValue((*Client).Features),
		"luci_getInitList.json": func(client *Client, ctx context.Context) error {
			_, err := client.InitList(ctx, "")

			return err
		},
		"luci_getLEDs.json":          fixtureCheckValue((*Client).LEDs),
		"luci_getMountPoints.json":   fixtureCheckValue((*Client).MountPoints),
		"luci_getOdhcp6cStats.json":  fixtureCheckValue((*Client).Odhcp6cStats),
		"luci_getProcessList.json":   fixtureCheckValue((*Client).Processes),
		"luci_getTimezones.json":     fixtureCheckValue((*Client).Timezones),
		"luci_getUnixtime.json":      fixtureCheckValue((*Client).Time),
		"luci_getLocaltime.json":     fixtureCheckValue((*Client).Time),
		"luci_getUSBDevices.json":    fixtureCheckValue((*Client).USBDevices),
		"luci_getVersion.json":       fixtureCheckValue((*Client).Version),
		"luci_rpc_getBoardJSON.json": fixtureCheckValue((*Client).BoardJSON),
		"luci_rpc_getDHCPLeases.json": func(client *Client, ctx context.Context) error {
			_, err := client.DHCPLeases(ctx, 0)

			return err
		},
		"luci_rpc_getDUIDHints.json":       fixtureCheckValue((*Client).DUIDHints),
		"luci_rpc_getHostHints.json":       fixtureCheckValue((*Client).HostHints),
		"luci_rpc_getNetworkDevices.json":  fixtureCheckValue((*Client).NetworkDevices),
		"luci_rpc_getWirelessDevices.json": fixtureCheckValue((*Client).WirelessDevices),
	}
}

func fixtureCheckValue[T any](call func(*Client, context.Context) (T, error)) fixtureCheck {
	return func(client *Client, ctx context.Context) error {
		_, err := call(client, ctx)

		return err
	}
}

func TestAdaptiveTimeMethod(t *testing.T) {
	for _, unavailable := range []error{goubus.ErrMethodNotFound, goubus.ErrInvalidCommand, goubus.ErrPermissionDenied} {
		t.Run(unavailable.Error(), func(t *testing.T) {
			var calls []string
			caller := goubus.CallerFunc(func(_ context.Context, _, method string, _ any) (goubus.Result, error) {
				calls = append(calls, method)
				if method == "getUnixtime" {
					return nil, unavailable
				}

				return goubus.NewJSONResult([]byte(`{"result":1737109342}`)), nil
			})
			client := New(caller)
			for range 2 {
				got, err := client.Time(context.Background())
				if err != nil || got.Unix() != 1737109342 {
					t.Fatalf("Time() = %v, %v", got, err)
				}
			}
			if len(calls) != 3 || calls[0] != "getUnixtime" || calls[1] != "getLocaltime" || calls[2] != "getLocaltime" {
				t.Fatalf("calls = %v", calls)
			}
		})
	}
}

func TestTimeMethodOverrideAndFailedCallsDoNotCache(t *testing.T) {
	var calls []string
	caller := goubus.CallerFunc(func(_ context.Context, _, method string, _ any) (goubus.Result, error) {
		calls = append(calls, method)
		if method == "getUnixtime" {
			return nil, goubus.ErrPermissionDenied
		}
		if len(calls) == 2 {
			return nil, errTemporaryLocaltimeFailure
		}

		return goubus.NewJSONResult([]byte(`{"result":100}`)), nil
	})
	client := New(caller)
	_, err := client.Time(context.Background())
	if err == nil {
		t.Fatal("expected first time lookup to fail")
	}
	_, err = client.Time(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 || calls[0] != "getUnixtime" || calls[1] != "getLocaltime" ||
		calls[2] != "getUnixtime" || calls[3] != "getLocaltime" {
		t.Fatalf("failed lookup polluted cache: %v", calls)
	}

	calls = nil
	fixed := New(caller, WithTimeMethod("getLocaltime"))
	_, err = fixed.Time(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0] != "getLocaltime" {
		t.Fatalf("fixed method calls = %v", calls)
	}
}

func TestFeatureAndProcessSchemas(t *testing.T) {
	featuresRaw := readLuciFixture(t, "rax3000m", "luci_getFeatures.json")
	features, err := fixtureLuciClient(featuresRaw).Features(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !features["odhcpd"].Enabled || features["odhcpd"].Subfeatures["ra"] != true {
		t.Fatalf("odhcpd feature = %+v", features["odhcpd"])
	}
	if features["bonding"].Enabled {
		t.Fatalf("null bonding feature unexpectedly enabled: %+v", features["bonding"])
	}
	x86Raw := readLuciFixture(t, "x86_generic", "luci_getFeatures.json")
	x86Features, err := fixtureLuciClient(x86Raw).Features(context.Background())
	if err != nil || !x86Features["odhcpd"].Enabled {
		t.Fatalf("scalar odhcpd = %+v, %v", x86Features["odhcpd"], err)
	}
	processRaw := readLuciFixture(t, "x86_generic", "luci_getProcessList.json")
	processes, err := fixtureLuciClient(processRaw).Processes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(processes) == 0 || processes[0].VSZ != "1219m" || processes[0].CPU == "" || processes[0].Mem == "" {
		t.Fatalf("process schema lost string fields: %+v", processes[:1])
	}
}

func TestMountUSBBlockLEDWirelessBoardSchemas(t *testing.T) {
	t.Run("mounts_and_usb", testMountUSBFixtures)
	t.Run("blocks_and_leds", testBlockLEDFixtures)
	t.Run("wireless_and_board", testWirelessBoardFixtures)
}

func testMountUSBFixtures(t *testing.T) {
	raw := readLuciFixture(t, "rax3000m", "luci_getMountPoints.json")
	mounts, err := fixtureLuciClient(raw).MountPoints(context.Background())
	if err != nil || len(mounts) == 0 || mounts[0].Size == 0 {
		t.Fatalf("mount points = %+v, %v", mounts, err)
	}
	raw = readLuciFixture(t, "rax3000m", "luci_getUSBDevices.json")
	usb, err := fixtureLuciClient(raw).USBDevices(context.Background())
	if err != nil || usb == nil || len(usb.Ports) == 0 {
		t.Fatalf("USB data = %+v, %v", usb, err)
	}
}

func testBlockLEDFixtures(t *testing.T) {
	raw := readLuciFixture(t, "rax3000m", "luci_getBlockDevices.json")
	blocks, err := fixtureLuciClient(raw).BlockDevices(context.Background())
	if err != nil || blocks == nil {
		t.Fatalf("block devices = %#v, %v", blocks, err)
	}
	raw = readLuciFixture(t, "rax3000m", "luci_getLEDs.json")
	leds, err := fixtureLuciClient(raw).LEDs(context.Background())
	if err != nil || len(leds) == 0 || len(leds["blue:status"].Triggers) == 0 {
		t.Fatalf("LED data = %+v, %v", leds, err)
	}
}

func testWirelessBoardFixtures(t *testing.T) {
	raw := readLuciFixture(t, "rax3000m", "luci_rpc_getWirelessDevices.json")
	wireless, err := fixtureLuciClient(raw).WirelessDevices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	interfaces := wireless["radio0"].Interfaces
	if len(interfaces) == 0 || interfaces[0].IWInfo == nil {
		t.Fatalf("wireless interface detail = %+v", wireless["radio0"])
	}
	raw = readLuciFixture(t, "rax3000m", "luci_rpc_getBoardJSON.json")
	board, err := fixtureLuciClient(raw).BoardJSON(context.Background())
	if err != nil || len(board.Network.LAN.Ports) == 0 {
		t.Fatalf("board LAN ports = %+v, %v", board, err)
	}
}

func fixtureLuciClient(raw []byte, options ...Option) *Client {
	caller := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return goubus.NewJSONResult(raw), nil
	})

	return New(caller, options...)
}

func readLuciFixture(t *testing.T, device, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "testdata", device, name)
	// #nosec G304 -- callers use only repository fixture names in fixed testdata directories.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return data
}
