package goubus

import (
	"encoding/json"
	"errors"
	"strings"
)

type UbusFileList struct {
	Entries []UbusFileListData
}

type UbusFileListData struct {
	Name string
	Type string
}

type UbusFileStat struct {
	Path  string
	Type  string
	Size  int
	Mode  int
	Atime int
	Mtime int
	Ctime int
	Inode int
	Uid   int
	Gid   int
}

type UbusFile struct {
	Data string
}

// FileExec executes a command on the remote system.
func (u *Client) FileExec(command string, params []string) (UbusExec, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusExec{}, errLogin
	}

	execData := map[string]interface{}{
		"command": command,
		"params":  params,
	}

	jsonStr := u.buildUbusCall("file", "exec", execData)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusExec{}, err
	}

	ubusData := UbusExec{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusExec{}, errors.New("data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// FileWrite writes data to a file on the remote system.
func (u *Client) FileWrite(path, data string, append bool, mode int, base64 bool) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	writeData := map[string]interface{}{
		"path":   path,
		"data":   data,
		"append": append,
		"mode":   mode,
		"base64": base64,
	}

	jsonStr := u.buildUbusCall("file", "write", writeData)
	_, err := u.Call(jsonStr)
	return err
}

// FileRead reads the contents of a file on the remote system.
func (u *Client) FileRead(path string) (UbusFile, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusFile{}, errLogin
	}

	readData := map[string]string{
		"path": path,
	}

	jsonStr := u.buildUbusCall("file", "read", readData)
	call, err := u.Call(jsonStr)
	if err != nil {
		if strings.Contains(err.Error(), "Object not found") {
			return UbusFile{}, errors.New("file module not found, try 'opkg update && opkg install rpcd-mod-file && service rpcd restart'")
		}
		return UbusFile{}, err
	}

	ubusData := UbusFile{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusFile{}, errors.New("data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// FileStat gets file/directory statistics on the remote system.
func (u *Client) FileStat(path string) (UbusFileStat, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusFileStat{}, errLogin
	}

	statData := map[string]string{
		"path": path,
	}

	jsonStr := u.buildUbusCall("file", "stat", statData)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusFileStat{}, err
	}

	ubusData := UbusFileStat{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusFileStat{}, errors.New("data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// FileList lists the contents of a directory on the remote system.
func (u *Client) FileList(path string) (UbusFileList, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusFileList{}, errLogin
	}

	listData := map[string]string{
		"path": path,
	}

	jsonStr := u.buildUbusCall("file", "list", listData)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusFileList{}, err
	}

	ubusData := UbusFileList{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusFileList{}, errors.New("data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}
