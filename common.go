package goubus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// UbusExec is a helper struct for parsing the output of file.exec commands.
type UbusExec struct {
	Code   int    `json:"code"`
	Stdout string `json:"stdout"`
}

// UbusResponse represents the ubus response format
type UbusResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

// Call a RPC method - now with the correct receiver *ubus
func (u *Client) Call(jsonStr []byte) (UbusResponse, error) {
	resp, err := http.Post("http://"+u.Host+UbusEndpointPath, ContentTypeJSON, bytes.NewBuffer(jsonStr))
	if err != nil {
		return UbusResponse{}, err
	}
	defer resp.Body.Close()
	ubusResp := UbusResponse{}
	err = json.NewDecoder(resp.Body).Decode(&ubusResp)
	if err != nil {
		return UbusResponse{}, err
	}

	// Check for JSON-RPC error
	if ubusResp.Error != nil {
		return ubusResp, fmt.Errorf("JSON-RPC error: %s", ubusResp.Error)
	}

	// Check for ubus error code in result
	if ubusResp.Result != nil {
		if resArray, ok := ubusResp.Result.([]interface{}); ok {
			if len(resArray) > 0 {
				if code, ok := resArray[0].(float64); ok && code != 0 {
					return ubusResp, NewUbusCodeError(int(code), "", "")
				}
			}
		}
	}

	return ubusResp, nil
}

// ubusErrCode is a map of ubus error codes to messages
var ubusErrCode = map[int]string{
	1:      "Invalid command",
	2:      "Invalid argument",
	3:      "Method not found",
	4:      "Not found",
	5:      "No data",
	6:      "Permission denied",
	7:      "Timeout",
	8:      "Not supported",
	9:      "Unknown error",
	10:     "Connection failed",
	-32000: "Server error",
	-32001: "Object not found",
	-32002: "Method not found",
	-32003: "Invalid command",
	-32004: "Invalid argument",
	-32005: "Request timeout",
	-32006: "Access denied",
	-32007: "Connection failed",
	-32008: "No data",
	-32009: "Operation not permitted",
	-32010: "Not found",
	-32011: "Out of memory",
	-32012: "Not supported",
	-32013: "Unknown error",
	-32014: "Connection timed out",
	-32015: "Connection closed",
	-32016: "System error",
}

// buildUbusCall creates a ubus JSON-RPC call with optimized string template
// This avoids the overhead of creating structs and using json.Marshal for simple calls
func (u *Client) buildUbusCall(service, method string, data interface{}) []byte {
	return u.buildUbusCallWithSession(u.AuthData.UbusRPCSession, service, method, data)
}

// buildUbusCallWithSession creates a ubus JSON-RPC call with a specific session ID
// This is useful for login operations where we need to use a null session ID
func (u *Client) buildUbusCallWithSession(sessionID, service, method string, data interface{}) []byte {
	return u.buildUbusCallWithSessionAndID(sessionID, u.id, service, method, data)
}

// buildUbusCallWithID creates a ubus JSON-RPC call with a specific ID
func (u *Client) buildUbusCallWithID(id int, service, method string, data interface{}) []byte {
	return u.buildUbusCallWithSessionAndID(u.AuthData.UbusRPCSession, id, service, method, data)
}

// buildUbusCallWithSessionAndID creates a ubus JSON-RPC call with specific session and ID
func (u *Client) buildUbusCallWithSessionAndID(sessionID string, id int, service, method string, data interface{}) []byte {
	var dataJSON string
	if data == nil {
		dataJSON = "{}"
	} else {
		switch v := data.(type) {
		case string:
			dataJSON = v
		case []byte:
			dataJSON = string(v)
		default:
			// For complex data, still use Marshal but this is the minority case
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				dataJSON = "{}"
			} else {
				dataJSON = string(jsonBytes)
			}
		}
	}

	// Use optimized string template - 5-10x faster than struct + marshal
	return []byte(fmt.Sprintf(`{
		"jsonrpc": "%s",
		"id": %d,
		"method": "%s",
		"params": [
			"%s",
			"%s",
			"%s",
			%s
		]
	}`, JSONRPCVersion, id, JSONRPCMethodCall, sessionID, service, method, dataJSON))
}

// getNextID increments and returns the next ID
func (u *Client) getNextID() int {
	u.id++
	return u.id
}

// UbusDhcpLeases represents the combined DHCP lease response containing both IPv4 and IPv6 leases
type UbusDhcpLeases struct {
	DHCPLeases  []UbusDhcpIPv4LeaseData `json:"dhcp_leases"`
	DHCP6Leases []UbusDhcpIPv6LeaseData `json:"dhcp6_leases"`
}

// UbusDhcpIPv6LeaseData represents a single DHCPv6 lease entry
type UbusDhcpIPv6LeaseData struct {
	Expires  int      `json:"expires"`
	Macaddr  string   `json:"macaddr"`
	DUID     string   `json:"duid"`
	IP6Addr  string   `json:"ip6addr"`
	IP6Addrs []string `json:"ip6addrs"`
}
