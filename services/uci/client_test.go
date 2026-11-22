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

func optionListCaller(t *testing.T, written *[]string) goubus.Caller {
	t.Helper()

	return goubus.CallerFunc(func(_ context.Context, _, method string, data any) (goubus.Result, error) {
		switch method {
		case "get":
			return goubus.NewJSONResult([]byte(`{"value":["server with spaces","other"]}`)), nil
		case "set":
			request, ok := data.(Request)
			if !ok {
				t.Fatalf("request has type %T, want uci.Request", data)
			}
			values, ok := request.Values["server"].([]string)
			if !ok {
				t.Fatalf("server values have type %T, want []string", request.Values["server"])
			}
			*written = values

			return goubus.NewJSONResult([]byte(`{}`)), nil
		default:
			return goubus.NewJSONResult([]byte(`{}`)), nil
		}
	})
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
