package api

import (
	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

// File service and method constants
const (
	ServiceFile      = "file"
	FileMethodRead   = "read"
	FileMethodWrite  = "write"
	FileMethodExec   = "exec"
	FileMethodStat   = "stat"
	FileMethodList   = "list"
	FileMethodMD5    = "md5"
	FileMethodRemove = "remove"
)

// File parameter constants
const (
	FileParamPath    = "path"
	FileParamData    = "data"
	FileParamAppend  = "append"
	FileParamMode    = "mode"
	FileParamBase64  = "base64"
	FileParamCommand = "command"
	FileParamParams  = "params"
	FileParamEnv     = "env"
)

// ReadFile reads the contents of a file on the remote system.
func ReadFile(caller types.Transport, path string, base64 bool) (*types.FileRead, error) {
	readData := map[string]any{
		FileParamPath: path,
	}
	if base64 {
		readData[FileParamBase64] = true
	}
	resp, err := caller.Call(ServiceFile, FileMethodRead, readData)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, errdefs.Wrapf(err, "file module not found or file does not exist at %s", path)
		}
		return nil, err
	}
	var fileRead types.FileRead
	if err := resp.Unmarshal(&fileRead); err != nil {
		return nil, err
	}
	return &fileRead, nil
}

// WriteFile writes data to a file on the remote system.
func WriteFile(caller types.Transport, path, data string, append bool, mode int, base64 bool) error {
	writeData := map[string]any{
		FileParamPath: path,
		FileParamData: data,
	}
	if append {
		writeData[FileParamAppend] = true
	}
	if mode != 0 {
		writeData[FileParamMode] = mode
	}
	if base64 {
		writeData[FileParamBase64] = true
	}
	_, err := caller.Call(ServiceFile, FileMethodWrite, writeData)
	return err
}

// ListDirectory lists the contents of a directory on the remote system.
func ListDirectory(caller types.Transport, path string) (*types.FileList, error) {
	listData := map[string]any{FileParamPath: path}
	resp, err := caller.Call(ServiceFile, FileMethodList, listData)
	if err != nil {
		return nil, err
	}
	var fileList types.FileList
	if err := resp.Unmarshal(&fileList); err != nil {
		return nil, err
	}
	return &fileList, nil
}

// StatFile gets file/directory statistics on the remote system.
func StatFile(caller types.Transport, path string) (*types.FileStat, error) {
	statData := map[string]any{FileParamPath: path}
	resp, err := caller.Call(ServiceFile, FileMethodStat, statData)
	if err != nil {
		return nil, err
	}
	var fileStat types.FileStat
	if err := resp.Unmarshal(&fileStat); err != nil {
		return nil, err
	}
	return &fileStat, nil
}

// FileMD5Response represents the response from a MD5 call.
type FileMD5Response struct {
	MD5 string `json:"md5"`
}

// GetFileMD5 calculates the MD5 checksum of a file.
func GetFileMD5(caller types.Transport, path string) (*string, error) {
	md5Data := map[string]any{FileParamPath: path}
	resp, err := caller.Call(ServiceFile, FileMethodMD5, md5Data)
	if err != nil {
		return nil, err
	}
	var fileMD5 FileMD5Response
	if err := resp.Unmarshal(&fileMD5); err != nil {
		return nil, err
	}
	return &fileMD5.MD5, nil
}

// RemoveFile removes a file or directory.
func RemoveFile(caller types.Transport, path string) error {
	removeData := map[string]any{FileParamPath: path}
	_, err := caller.Call(ServiceFile, FileMethodRemove, removeData)
	return err
}

// ExecuteCommand executes a command on the remote system.
func ExecuteCommand(caller types.Transport, command string, params []string, env map[string]string) (*types.FileExec, error) {
	execData := map[string]any{
		FileParamCommand: command,
		FileParamParams:  params,
		FileParamEnv:     env,
	}
	resp, err := caller.Call(ServiceFile, FileMethodExec, execData)
	if err != nil {
		return nil, err
	}
	var fileExec types.FileExec
	if err := resp.Unmarshal(&fileExec); err != nil {
		return nil, err
	}
	return &fileExec, nil
}
