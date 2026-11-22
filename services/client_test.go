package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/services/dhcp"
	"github.com/honeybbq/goubus/v2/services/luci"
)

func TestNewPanicsOnNilCaller(t *testing.T) {
	defer func() {
		if got := recover(); got != "goubus/services: nil caller" {
			t.Fatalf("panic = %v", got)
		}
	}()
	New(nil)
}

func TestAccessorsReturnStableClients(t *testing.T) {
	caller := goubus.CallerFunc(func(context.Context, string, string, any) (goubus.Result, error) {
		return goubus.NewJSONResult(json.RawMessage(`{}`)), nil
	})
	client := New(caller)
	accessors := []struct {
		get  func() any
		name string
	}{
		{get: func() any { return client.Container() }, name: "Container"},
		{get: func() any { return client.DHCP() }, name: "DHCP"},
		{get: func() any { return client.File() }, name: "File"},
		{get: func() any { return client.Hostapd() }, name: "Hostapd"},
		{get: func() any { return client.Log() }, name: "Log"},
		{get: func() any { return client.LuCI() }, name: "LuCI"},
		{get: func() any { return client.Network() }, name: "Network"},
		{get: func() any { return client.RC() }, name: "RC"},
		{get: func() any { return client.RPCSys() }, name: "RPCSys"},
		{get: func() any { return client.Service() }, name: "Service"},
		{get: func() any { return client.Session() }, name: "Session"},
		{get: func() any { return client.System() }, name: "System"},
		{get: func() any { return client.UCI() }, name: "UCI"},
		{get: func() any { return client.Wireless() }, name: "Wireless"},
		{get: func() any { return client.WPASupplicant() }, name: "WPASupplicant"},
	}
	for _, accessor := range accessors {
		t.Run(accessor.name, func(t *testing.T) {
			first := accessor.get()
			if first == nil {
				t.Fatal("accessor returned nil")
			}
			if first != accessor.get() {
				t.Fatal("accessor did not return the same client pointer")
			}
		})
	}
}

func TestWithDHCPAndLuCIOptions(t *testing.T) {
	var (
		calls      int
		dhcpMethod string
		dhcpParams any
		luciMethod string
	)
	caller := goubus.CallerFunc(func(_ context.Context, service, method string, params any) (goubus.Result, error) {
		calls++
		if service == "dhcp" {
			dhcpMethod = method
			dhcpParams = params
		}
		if service == "luci" {
			luciMethod = method
		}

		return goubus.NewJSONResult(json.RawMessage(`{"result":123}`)), nil
	})
	client := New(caller,
		WithDHCP(dhcp.WithDUIDFormat(dhcp.DUIDString)),
		WithLuCI(luci.WithTimeMethod("getLocaltime")),
	)

	err := client.DHCP().AddLease(context.Background(), dhcp.AddLeaseRequest{DUID: "duid"})
	if err != nil {
		t.Fatal(err)
	}
	gotTime, err := client.LuCI().Time(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Unix(123, 0); !gotTime.Equal(want) {
		t.Errorf("Time() = %v, want %v", gotTime, want)
	}
	if calls != 2 {
		t.Fatalf("call count = %d, want 2", calls)
	}
	if dhcpMethod != "add_lease" {
		t.Errorf("DHCP method = %q, want add_lease", dhcpMethod)
	}
	params, ok := dhcpParams.(map[string]any)
	if !ok || params["duid"] != "duid" {
		t.Errorf("DHCP params = %#v, want string DUID", dhcpParams)
	}
	if luciMethod != "getLocaltime" {
		t.Errorf("LuCI method = %q, want getLocaltime", luciMethod)
	}
}
