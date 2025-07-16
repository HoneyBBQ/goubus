package goubus

// File returns a manager for file operations on the device.
func (c *Client) File() *FileManager {
	return &FileManager{
		client: c,
	}
}

// FileManager provides methods to interact with the device's filesystem via ubus.
type FileManager struct {
	client *Client
}

// Read reads the contents of a file.
func (fm *FileManager) Read(path string) (UbusFile, error) {
	return fm.client.fileRead(path)
}

// Write writes data to a file.
func (fm *FileManager) Write(path, data string, append bool, mode int, base64 bool) error {
	return fm.client.fileWrite(path, data, append, mode, base64)
}

// List lists the contents of a directory.
func (fm *FileManager) List(path string) (UbusFileList, error) {
	return fm.client.fileList(path)
}

// Stat gets file/directory status information.
func (fm *FileManager) Stat(path string) (UbusFileStat, error) {
	return fm.client.fileStat(path)
}

// Exec executes a command and returns the result.
func (fm *FileManager) Exec(command string, params []string) (UbusExec, error) {
	return fm.client.fileExec(command, params)
}

// Delete removes a file or directory from the filesystem.
func (fm *FileManager) Delete(path string) error {
	// Use the 'rm' command to delete files/directories
	_, err := fm.Exec("rm", []string{"-rf", path})
	return err
}

// Chmod changes the permissions of a file or directory.
func (fm *FileManager) Chmod(path string, mode string) error {
	_, err := fm.Exec("chmod", []string{mode, path})
	return err
}

// Chown changes the owner and group of a file or directory.
func (fm *FileManager) Chown(path string, owner string) error {
	_, err := fm.Exec("chown", []string{owner, path})
	return err
}

// Mkdir creates a directory (and parent directories if needed).
func (fm *FileManager) Mkdir(path string, createParents bool) error {
	args := []string{}
	if createParents {
		args = append(args, "-p")
	}
	args = append(args, path)
	_, err := fm.Exec("mkdir", args)
	return err
}

// Copy copies a file or directory from source to destination.
func (fm *FileManager) Copy(source, destination string) error {
	_, err := fm.Exec("cp", []string{"-r", source, destination})
	return err
}

// Move moves/renames a file or directory from source to destination.
func (fm *FileManager) Move(source, destination string) error {
	_, err := fm.Exec("mv", []string{source, destination})
	return err
}

// CreateSymlink creates a symbolic link from target to linkname.
func (fm *FileManager) CreateSymlink(target, linkname string) error {
	_, err := fm.Exec("ln", []string{"-s", target, linkname})
	return err
}

// CreateHardlink creates a hard link from target to linkname.
func (fm *FileManager) CreateHardlink(target, linkname string) error {
	_, err := fm.Exec("ln", []string{target, linkname})
	return err
}

// Touch creates an empty file or updates the timestamp of an existing file.
func (fm *FileManager) Touch(path string) error {
	_, err := fm.Exec("touch", []string{path})
	return err
}

// Find searches for files and directories matching the given pattern.
func (fm *FileManager) Find(basePath, pattern string) (UbusExec, error) {
	return fm.Exec("find", []string{basePath, "-name", pattern})
}

// Grep searches for text patterns in files.
func (fm *FileManager) Grep(pattern, path string, recursive bool) (UbusExec, error) {
	args := []string{}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, pattern, path)
	return fm.Exec("grep", args)
}

// Tar creates or extracts tar archives.
func (fm *FileManager) Tar(operation string, archivePath string, files []string) (UbusExec, error) {
	args := []string{operation, archivePath}
	args = append(args, files...)
	return fm.Exec("tar", args)
}

// DiskUsage gets disk usage information for a path.
func (fm *FileManager) DiskUsage(path string, humanReadable bool) (UbusExec, error) {
	args := []string{}
	if humanReadable {
		args = append(args, "-h")
	}
	args = append(args, path)
	return fm.Exec("du", args)
}
