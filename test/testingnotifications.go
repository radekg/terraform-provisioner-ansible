package test

// NotificationDirectoryCreated is a test SFTP file system dirctory created notification.
type NotificationDirectoryCreated struct {
	MemFile *memFile
}

// NotificationDirectoryDeleted is a test SFTP file system dirctory deleted notification.
type NotificationDirectoryDeleted struct {
	MemFile *memFile
}

// NotificationSymlinkCreated is a test SFTP file system symlink created notification.
type NotificationSymlinkCreated struct {
	MemFile *memFile
}

// NotificationFileRenamed is a test SFTP file system file renamed notification.
type NotificationFileRenamed struct {
	MemFile    *memFile
	SourcePath string
}

// NotificationFileWritten is a test SFTP file system file written notification.
type NotificationFileWritten struct {
	MemFile *memFile
}

// NotificationCommandExecuted is a test SSH server command execution notification.
type NotificationCommandExecuted struct {
	Command string
}
