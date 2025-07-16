package goubus

import (
	"time"
)

// SystemInfo holds runtime system information from 'ubus call system info'.
type SystemInfo struct {
	LocalTime time.Time `json:"localtime"`
	Uptime    int       `json:"uptime"`
	Load      []int     `json:"load"`
	Memory    Memory    `json:"memory"`
	Root      Storage   `json:"root"`
	Tmp       Storage   `json:"tmp"`
	Swap      Swap      `json:"swap"`
}

// SystemBoardInfo holds hardware-specific information from 'ubus call system board'.
type SystemBoardInfo struct {
	Kernel     string  `json:"kernel"`
	Hostname   string  `json:"hostname"`
	System     string  `json:"system"`
	Model      string  `json:"model"`
	BoardName  string  `json:"board_name"`
	RootfsType string  `json:"rootfs_type"`
	Release    Release `json:"release"`
}

// Release holds release information.
type Release struct {
	Distribution string `json:"distribution"`
	Version      string `json:"version"`
	Revision     string `json:"revision"`
	Codename     string `json:"codename"`
	Target       string `json:"target"`
	Description  string `json:"description"`
}

// Memory holds memory usage statistics.
type Memory struct {
	Total     int `json:"total"`
	Free      int `json:"free"`
	Shared    int `json:"shared"`
	Buffered  int `json:"buffered"`
	Available int `json:"available"`
	Cached    int `json:"cached"`
}

// Storage holds storage usage statistics.
type Storage struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
	Avail int `json:"avail"`
}

// Swap holds swap usage statistics.
type Swap struct {
	Total int `json:"total"`
	Free  int `json:"free"`
}

func (u *Client) callSystem(method string) (map[string]interface{}, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return nil, errLogin
	}
	jsonStr := u.buildUbusCall(ServiceSystem, method, nil)
	call, err := u.Call(jsonStr)
	if err != nil {
		return nil, err
	}
	if len(call.Result.([]interface{})) < 2 {
		return nil, ErrInvalidResponse
	}
	if result, ok := call.Result.([]interface{})[1].(map[string]interface{}); ok {
		return result, nil
	}
	return nil, NewError(ErrorCodeUnexpectedFormat, "unexpected type for system result")
}

// systemGetInfo retrieves runtime system information (private method).
func (u *Client) systemGetInfo() (*SystemInfo, error) {
	// Get runtime information
	infoData, err := u.callSystem(MethodInfo)
	if err != nil {
		return nil, err
	}

	info := &SystemInfo{}

	// Extract runtime information from info data
	if val, ok := infoData["uptime"].(float64); ok {
		info.Uptime = int(val)
	}
	if lt, ok := infoData["localtime"].(float64); ok {
		info.LocalTime = time.Unix(int64(lt), 0)
	}
	if load, ok := infoData["load"].([]interface{}); ok {
		for _, v := range load {
			if l, ok := v.(float64); ok {
				info.Load = append(info.Load, int(l))
			}
		}
	}
	if mem, ok := infoData["memory"].(map[string]interface{}); ok {
		if v, ok := mem["total"].(float64); ok {
			info.Memory.Total = int(v)
		}
		if v, ok := mem["free"].(float64); ok {
			info.Memory.Free = int(v)
		}
		if v, ok := mem["shared"].(float64); ok {
			info.Memory.Shared = int(v)
		}
		if v, ok := mem["buffered"].(float64); ok {
			info.Memory.Buffered = int(v)
		}
		if v, ok := mem["available"].(float64); ok {
			info.Memory.Available = int(v)
		}
		if v, ok := mem["cached"].(float64); ok {
			info.Memory.Cached = int(v)
		}
	}
	if root, ok := infoData["root"].(map[string]interface{}); ok {
		if v, ok := root["total"].(float64); ok {
			info.Root.Total = int(v)
		}
		if v, ok := root["free"].(float64); ok {
			info.Root.Free = int(v)
		}
		if v, ok := root["used"].(float64); ok {
			info.Root.Used = int(v)
		}
		if v, ok := root["avail"].(float64); ok {
			info.Root.Avail = int(v)
		}
	}
	if tmp, ok := infoData["tmp"].(map[string]interface{}); ok {
		if v, ok := tmp["total"].(float64); ok {
			info.Tmp.Total = int(v)
		}
		if v, ok := tmp["free"].(float64); ok {
			info.Tmp.Free = int(v)
		}
		if v, ok := tmp["used"].(float64); ok {
			info.Tmp.Used = int(v)
		}
		if v, ok := tmp["avail"].(float64); ok {
			info.Tmp.Avail = int(v)
		}
	}
	if swap, ok := infoData["swap"].(map[string]interface{}); ok {
		if v, ok := swap["total"].(float64); ok {
			info.Swap.Total = int(v)
		}
		if v, ok := swap["free"].(float64); ok {
			info.Swap.Free = int(v)
		}
	}

	return info, nil
}

// systemGetBoardInfo retrieves hardware-specific board information (private method).
func (u *Client) systemGetBoardInfo() (*SystemBoardInfo, error) {
	boardData, err := u.callSystem(MethodBoard)
	if err != nil {
		return nil, err
	}

	boardInfo := &SystemBoardInfo{}

	// Extract hardware information from board data
	if val, ok := boardData["hostname"].(string); ok {
		boardInfo.Hostname = val
	}
	if val, ok := boardData["model"].(string); ok {
		boardInfo.Model = val
	}
	if val, ok := boardData["system"].(string); ok {
		boardInfo.System = val
	}
	if val, ok := boardData["kernel"].(string); ok {
		boardInfo.Kernel = val
	}
	if val, ok := boardData["board_name"].(string); ok {
		boardInfo.BoardName = val
	}
	if val, ok := boardData["rootfs_type"].(string); ok {
		boardInfo.RootfsType = val
	}

	// Extract release information from board data
	if rel, ok := boardData["release"].(map[string]interface{}); ok {
		if v, ok := rel["distribution"].(string); ok {
			boardInfo.Release.Distribution = v
		}
		if v, ok := rel["version"].(string); ok {
			boardInfo.Release.Version = v
		}
		if v, ok := rel["revision"].(string); ok {
			boardInfo.Release.Revision = v
		}
		if v, ok := rel["codename"].(string); ok {
			boardInfo.Release.Codename = v
		}
		if v, ok := rel["target"].(string); ok {
			boardInfo.Release.Target = v
		}
		if v, ok := rel["description"].(string); ok {
			boardInfo.Release.Description = v
		}
	}

	return boardInfo, nil
}

// systemReboot reboots the device (private method).
func (u *Client) systemReboot() error {
	_, err := u.callSystem(MethodReboot)
	return err
}
