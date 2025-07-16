package goubus

import (
	"encoding/json"
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
func (u *Client) fileExec(command string, params []string) (UbusExec, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusExec{}, errLogin
	}

	execData := map[string]interface{}{
		ParamCommand: command,
		ParamParams:  params,
	}

	jsonStr := u.buildUbusCall(ServiceFile, MethodExec, execData)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusExec{}, err
	}

	ubusData := UbusExec{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusExec{}, ErrDataParsingError
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// FileWrite writes data to a file on the remote system.
func (u *Client) fileWrite(path, data string, append bool, mode int, base64 bool) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	writeData := map[string]interface{}{
		ParamPath:   path,
		ParamData:   data,
		ParamAppend: append,
		ParamMode:   mode,
		ParamBase64: base64,
	}

	jsonStr := u.buildUbusCall(ServiceFile, MethodWrite, writeData)
	_, err := u.Call(jsonStr)
	return err
}

// FileRead reads the contents of a file on the remote system.
func (u *Client) fileRead(path string) (UbusFile, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusFile{}, errLogin
	}

	readData := map[string]interface{}{
		ParamPath: path,
	}

	jsonStr := u.buildUbusCall(ServiceFile, MethodRead, readData)
	call, err := u.Call(jsonStr)
	if err != nil {
		if strings.Contains(err.Error(), "Object not found") {
			return UbusFile{}, ErrFileModuleNotFound
		}
		return UbusFile{}, err
	}

	ubusData := UbusFile{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusFile{}, ErrDataParsingError
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// FileStat gets file/directory statistics on the remote system.
func (u *Client) fileStat(path string) (UbusFileStat, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusFileStat{}, errLogin
	}

	statData := map[string]interface{}{
		ParamPath: path,
	}

	jsonStr := u.buildUbusCall(ServiceFile, MethodStat, statData)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusFileStat{}, err
	}

	ubusData := UbusFileStat{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusFileStat{}, ErrDataParsingError
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// FileList lists the contents of a directory on the remote system.
func (u *Client) fileList(path string) (UbusFileList, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusFileList{}, errLogin
	}

	listData := map[string]interface{}{
		ParamPath: path,
	}

	jsonStr := u.buildUbusCall(ServiceFile, MethodList, listData)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusFileList{}, err
	}

	ubusData := UbusFileList{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusFileList{}, ErrDataParsingError
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}
