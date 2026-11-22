package uci

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func TestConfigsFixtures(t *testing.T) {
	for _, device := range []string{"rax3000m", "x86_generic"} {
		// #nosec G304 -- the profile comes from the fixed test profile list.
		raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "testdata", device, "uci_configs.json"))
		if err != nil {
			t.Fatalf("read %s configs: %v", device, err)
		}
		configs, err := fixtureUCIClient(raw).Configs(context.Background())
		if err != nil {
			t.Fatalf("Configs %s: %v", device, err)
		}
		if len(configs) == 0 || !slices.Contains(configs, "network") {
			t.Fatalf("Configs %s = %v", device, configs)
		}
	}
}

func TestPackageFixtures(t *testing.T) {
	ctx := context.Background()
	for _, device := range []string{"rax3000m", "x86_generic"} {
		pattern := filepath.Join("..", "..", "internal", "testdata", device, "uci_get_*.json")
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) == 0 {
			t.Fatalf("no UCI fixtures matched %s", pattern)
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
			var outer struct {
				Values map[string]json.RawMessage `json:"values"`
			}
			err = json.Unmarshal(raw, &outer)
			if err != nil {
				t.Fatalf("decode %s: %v", file, err)
			}
			client := fixtureUCIClient(raw)
			packageContext := client.Package("fixture")
			sections, callErr := packageContext.All(ctx)
			if callErr != nil {
				t.Fatalf("All %s: %v", file, callErr)
			}
			if len(sections) != len(outer.Values) {
				t.Fatalf("All %s returned %d sections, want %d", file, len(sections), len(outer.Values))
			}
			names, callErr := packageContext.Sections(ctx)
			if callErr != nil {
				t.Fatalf("Sections %s: %v", file, callErr)
			}
			if len(names) != len(outer.Values) || !slices.IsSorted(names) {
				t.Fatalf("Sections %s = %v, want all sorted values sections", file, names)
			}
		}
	}
}

func TestOptionListValuesPreserveSpaces(t *testing.T) {
	ctx := context.Background()
	var written []string
	client := New(optionListCaller(t, &written))
	option := client.Package("system").Section("ntp").Option("server")
	first, err := option.Get(ctx)
	if err != nil || first != "server with spaces" {
		t.Fatalf("Get() = %q, %v", first, err)
	}
	values, err := option.Values(ctx)
	if err != nil || len(values) != 2 || values[0] != "server with spaces" {
		t.Fatalf("Values() = %v, %v", values, err)
	}
	err = option.AddToList(ctx, "new host value")
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 3 || written[0] != "server with spaces" || written[2] != "new host value" {
		t.Fatalf("AddToList wrote %#v", written)
	}
	err = option.DeleteFromList(ctx, "server with spaces")
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 || written[0] != "other" {
		t.Fatalf("DeleteFromList wrote %#v", written)
	}
}

func TestOptionStateValues(t *testing.T) {
	option := New(optionListCaller(t, new([]string))).Package("system").Section("ntp").Option("server")
	values, err := option.StateValues(context.Background())
	if err != nil || len(values) != 2 || values[1] != "other" {
		t.Fatalf("StateValues() = %v, %v", values, err)
	}
}

func optionListCaller(t *testing.T, written *[]string) goubus.Caller {
	t.Helper()

	return goubus.CallerFunc(func(_ context.Context, _, method string, data any) (goubus.Result, error) {
		switch method {
		case "get", "state":
			return goubus.NewJSONResult([]byte(`{"value":["server with spaces","other"]}`)), nil
		case "set":
			writeReq, ok := data.(writeRequest)
			if !ok {
				t.Fatalf("request has type %T, want uci.writeRequest", data)
			}
			values, ok := writeReq.Values["server"].([]string)
			if !ok {
				t.Fatalf("server values have type %T, want []string", writeReq.Values["server"])
			}
			*written = values

			return goubus.NewJSONResult([]byte(`{}`)), nil
		default:
			return goubus.NewJSONResult([]byte(`{}`)), nil
		}
	})
}
func TestSectionValuesMarshalValueReceiver(t *testing.T) {
	var zeroValue SectionValues
	zeroValue.Set("hostname", "router")
	encoded, err := json.Marshal(zeroValue)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"hostname":"router"}` {
		t.Fatalf("json.Marshal(zero-value SectionValues) = %s", encoded)
	}

	values := SectionValuesFromStrings(map[string]string{"scalar": "one"})
	values.SetList("list", "alpha", "beta")
	encoded, err = json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"list":["alpha","beta"],"scalar":"one"}` {
		t.Fatalf("json.Marshal(SectionValues) = %s", encoded)
	}
}

func TestMetadataAnonymousAcceptsJSONNumbers(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want bool
	}{
		{name: "one", raw: `{"values":{".anonymous":1}}`, want: true},
		{name: "zero", raw: `{"values":{".anonymous":0}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			section, err := fixtureUCIClient([]byte(tc.raw)).Package("network").Section("wifi").Get(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if bool(section.Metadata.Anonymous) != tc.want {
				t.Fatalf("Anonymous = %t, want %t", section.Metadata.Anonymous, tc.want)
			}
		})
	}
}

func TestMetadataAnonymousAcceptsSocketValues(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  bool
	}{
		{value: int64(1), want: true},
		{value: int64(0)},
		{value: float64(2), want: true},
		{value: float64(0)},
		{value: uint64(1), want: true},
		{value: uint64(0)},
		{value: true, want: true},
		{value: "1", want: true},
		{value: "true", want: true},
	} {
		if got := parseMetadata(map[string]any{".anonymous": tc.value}).Anonymous; bool(got) != tc.want {
			t.Errorf("parseMetadata(%T(%v)).Anonymous = %t, want %t", tc.value, tc.value, got, tc.want)
		}
	}
}

func TestGetResponseAcceptsScalarAndList(t *testing.T) {
	var response GetResponse
	err := json.Unmarshal([]byte(`{"value":"single value"}`), &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Value.First() != "single value" {
		t.Fatalf("scalar Value = %#v", response.Value)
	}
	err = json.Unmarshal([]byte(`{"value":["one value","two"]}`), &response)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Value) != 2 || response.Value[0] != "one value" {
		t.Fatalf("list Value = %#v", response.Value)
	}
}

func fixtureUCIClient(raw []byte) *Client {
	return New(goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return goubus.NewJSONResult(raw), nil
	}))
}
