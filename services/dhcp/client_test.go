package dhcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

var errUnrelatedDUIDFailure = errors.New("unrelated failure")

func TestLeaseFixtures(t *testing.T) {
	ctx := context.Background()
	fixtures := []struct {
		device string
		file   string
		method string
	}{
		{"rax3000m", "dhcp_ipv6leases.json", "ipv6leases"},
		{"rax3000m", "dhcp_ipv6ra.json", "ipv6ra"},
		{"x86_generic", "dhcp_ipv4leases.json", "ipv4leases"},
		{"x86_generic", "dhcp_ipv6leases.json", "ipv6leases"},
		{"x86_generic", "dhcp_ipv6ra.json", "ipv6ra"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.device+"/"+fixture.file, func(t *testing.T) {
			raw := readFixture(t, fixture.device, fixture.file)
			client := New(goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
				return goubus.NewJSONResult(raw), nil
			}))
			switch fixture.method {
			case "ipv4leases":
				leases, err := client.IPv4Leases(ctx)
				if err != nil {
					t.Fatalf("%s/%s: %v", fixture.device, fixture.file, err)
				}
				assertInterfaces(t, fixture.device, leaseInterfaces4(leases))
			case "ipv6leases":
				leases, err := client.IPv6Leases(ctx)
				if err != nil {
					t.Fatalf("%s/%s: %v", fixture.device, fixture.file, err)
				}
				assertInterfaces(t, fixture.device, leaseInterfaces6(leases))
			case "ipv6ra":
				entries, err := client.IPv6RA(ctx)
				if err != nil {
					t.Fatalf("%s/%s: %v", fixture.device, fixture.file, err)
				}
				assertInterfaces(t, fixture.device, raInterfaces(entries))
				if fixture.device == "x86_generic" && len(entries) > 0 && entries[0].Length != 60 {
					t.Fatalf("x86 prefix length = %d, want 60", entries[0].Length)
				}
			}
		})
	}
}

func TestIPv6RAStaleDecodesBlobmsgBooleanNumbers(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want bool
	}{
		{name: "true", raw: `{"stale":1}`, want: true},
		{name: "false", raw: `{"stale":0}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ra IPv6RA
			err := json.Unmarshal([]byte(tc.raw), &ra)
			if err != nil {
				t.Fatal(err)
			}
			if bool(ra.Stale) != tc.want {
				t.Fatalf("Stale = %t, want %t", ra.Stale, tc.want)
			}
		})
	}
}

func TestIPv6LeaseAliases(t *testing.T) {
	var lease IPv6Lease
	err := json.Unmarshal([]byte(`{"address":"2001:db8::1","duid":"d","iaid":"7","assigned":10,`+
		`"valid":20,"length":64}`), &lease)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Address != "2001:db8::1" || lease.DUID != "d" || lease.IAID != "7" || lease.Valid != 20 {
		t.Fatalf("unexpected IPv6 lease: %+v", lease)
	}
	err = json.Unmarshal([]byte(`{"ipv6-addr":["2001:db8::2"]}`), &lease)
	if err != nil {
		t.Fatal(err)
	}
	if len(lease.IPAddr) != 1 || lease.Address != lease.IPAddr[0] {
		t.Fatalf("IPv6 aliases not decoded: %+v", lease)
	}
}

func TestAddLeaseOmitsEmptyValuesAndDoesNotProbeWithoutDUID(t *testing.T) {
	calls, lists := 0, 0
	caller := listCaller{Caller: goubus.CallerFunc(func(_ context.Context, _, _ string, data any) (goubus.Result, error) {
		calls++
		params, ok := data.(map[string]any)
		if !ok {
			t.Errorf("request has type %T, want map[string]any", data)

			return goubus.NewJSONResult([]byte(`{}`)), nil
		}
		if len(params) != 0 {
			t.Errorf("empty request sent fields: %#v", params)
		}

		return goubus.NewJSONResult([]byte(`{}`)), nil
	}), onList: func() { lists++ }}
	err := New(caller).AddLease(context.Background(), AddLeaseRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || lists != 0 {
		t.Fatalf("calls=%d signature probes=%d, want 1 and 0", calls, lists)
	}
}

func TestAddLeaseSignatureAndExplicitFormats(t *testing.T) {
	runDUIDFormatTest(t, "signature-array", goubus.ArgTypeArray, nil, []string{"duid"})
	runDUIDFormatTest(t, "signature-string", goubus.ArgTypeString, nil, "duid")
	runDUIDFormatTest(t, "explicit-string", goubus.ArgTypeArray, WithDUIDFormat(DUIDString), "duid")
}

func runDUIDFormatTest(t *testing.T, name string, signature goubus.ArgType, option Option, want any) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		var got any
		lists := 0
		base := goubus.CallerFunc(func(_ context.Context, _, _ string, data any) (goubus.Result, error) {
			params, ok := data.(map[string]any)
			if !ok {
				t.Fatalf("request has type %T, want map[string]any", data)
			}
			got = params["duid"]

			return goubus.NewJSONResult([]byte(`{}`)), nil
		})
		caller := listCaller{Caller: base, signature: signature, onList: func() { lists++ }}
		client := New(caller, option)
		request := AddLeaseRequest{IP: "1.2.3.4", MAC: "aa", DUID: "duid"}
		err := client.AddLease(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if !equalDUID(got, want) {
			t.Fatalf("DUID argument = %#v, want %#v", got, want)
		}
		if option != nil && lists != 0 {
			t.Fatalf("explicit option triggered %d signature probes", lists)
		}
	})
}

func TestAddLeaseFallbackCachesOnlySuccessfulFormat(t *testing.T) {
	calls := 0
	var formats []any
	caller := goubus.CallerFunc(func(_ context.Context, _, _ string, data any) (goubus.Result, error) {
		calls++
		params, ok := data.(map[string]any)
		if !ok {
			t.Fatalf("request has type %T, want map[string]any", data)
		}
		duid := params["duid"]
		formats = append(formats, duid)
		switch calls {
		case 1:
			return nil, errUnrelatedDUIDFailure
		case 2:
			if _, ok := duid.([]string); !ok {
				t.Fatalf("second call did not retry array: %#v", duid)
			}

			return nil, goubus.ErrInvalidParameter
		case 3:
			return goubus.NewJSONResult([]byte(`{}`)), nil
		default:
			if duid != "duid" {
				t.Fatalf("cached format = %#v, want string", duid)
			}

			return goubus.NewJSONResult([]byte(`{}`)), nil
		}
	})
	client := New(caller)
	request := AddLeaseRequest{DUID: "duid"}
	err := client.AddLease(context.Background(), request)
	if err == nil {
		t.Fatal("first call should fail")
	}
	err = client.AddLease(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	err = client.AddLease(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 || len(formats) != 4 {
		t.Fatalf("calls=%d, want 4 including one cached call", calls)
	}
}

func readFixture(t *testing.T, device, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "testdata", device, name)
	// #nosec G304 -- fixture paths are built from fixed device and file names.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return data
}

func leaseInterfaces4(leases []IPv4Lease) []string {
	result := make([]string, len(leases))
	for index := range leases {
		result[index] = leases[index].Interface
	}

	return result
}

func leaseInterfaces6(leases []IPv6Lease) []string {
	result := make([]string, len(leases))
	for index := range leases {
		result[index] = leases[index].Interface
	}

	return result
}

func raInterfaces(entries []IPv6RA) []string {
	result := make([]string, len(entries))
	for index := range entries {
		result[index] = entries[index].Interface
	}

	return result
}

func assertInterfaces(t *testing.T, device string, interfaces []string) {
	t.Helper()
	for _, name := range interfaces {
		if name == "" {
			t.Fatalf("%s fixture returned an empty interface name", device)
		}
	}
}

func equalDUID(got, want any) bool {
	gotArray, isArray := got.([]string)
	wantArray, wantsArray := want.([]string)
	if isArray || wantsArray {
		return isArray && wantsArray && len(gotArray) == 1 && len(wantArray) == 1 && gotArray[0] == wantArray[0]
	}

	return got == want
}

type listCaller struct {
	goubus.Caller
	onList    func()
	signature goubus.ArgType
}

func (caller listCaller) List(context.Context, ...string) (map[string]goubus.ObjectSignatures, error) {
	if caller.onList != nil {
		caller.onList()
	}

	return map[string]goubus.ObjectSignatures{
		"dhcp": {"add_lease": {"duid": caller.signature}},
	}, nil
}
