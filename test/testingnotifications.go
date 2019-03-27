package test

type NotificationDirectoryCreated struct {
	MemFile *memFile
}

type NotificationDirectoryDeleted struct {
	MemFile *memFile
}

type NotificationSymlinkCreated struct {
	MemFile *memFile
}

type NotificationFileRenamed struct {
	MemFile    *memFile
	SourcePath string
}

type NotificationFileWritten struct {
	MemFile *memFile
}

type NotificationCommandExecuted struct {
	Command string
}
