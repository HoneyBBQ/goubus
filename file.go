package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// FileManager provides an interface for interacting with the device's filesystem.
type FileManager struct {
	client *Client
}

// File returns a new FileManager.
func (c *Client) File() *FileManager {
	return &FileManager{client: c}
}

// Read retrieves the contents of a file.
func (fm *FileManager) Read(path string, base64 bool) (*types.FileRead, error) {
	return api.ReadFile(fm.client.caller, path, base64)
}

// Write writes data to a file.
func (fm *FileManager) Write(path, data string, append bool, mode int, base64 bool) error {
	return api.WriteFile(fm.client.caller, path, data, append, mode, base64)
}

// List lists the contents of a directory.
func (fm *FileManager) List(path string) (*types.FileList, error) {
	return api.ListDirectory(fm.client.caller, path)
}

// Stat retrieves statistics for a file or directory.
func (fm *FileManager) Stat(path string) (*types.FileStat, error) {
	return api.StatFile(fm.client.caller, path)
}

// MD5 calculates the MD5 checksum of a file.
func (fm *FileManager) MD5(path string) (*string, error) {
	return api.GetFileMD5(fm.client.caller, path)
}

// Remove deletes a file or directory.
func (fm *FileManager) Remove(path string) error {
	return api.RemoveFile(fm.client.caller, path)
}

// Exec executes a command on the remote system.
func (fm *FileManager) Exec(command string, params []string, env map[string]string) (*types.FileExec, error) {
	return api.ExecuteCommand(fm.client.caller, command, params, env)
}
