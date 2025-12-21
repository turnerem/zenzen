package storage

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"

	"github.com/turnerem/zenzen/core"
)

// get all logs from fs
// add log
// delete log

const FILENAME = "notes.json"

type FSFileSystem struct {
	fileSystem fs.FS
}

func NewFSFileSystem(fileSystem fs.FS) *FSFileSystem {
	return &FSFileSystem{fileSystem: fileSystem}
}

func (o *FSFileSystem) GetAll() ([]core.Entry, error) {
	logs, err := getLogs(o.fileSystem, FILENAME)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func getLogs(fileSystem fs.FS, filename string) ([]core.Entry, error) {
	logFile, err := fs.ReadFile(fileSystem, filename)
	if err != nil {
		return nil, err
	}

	var logs []core.Entry
	decoder := json.NewDecoder(bytes.NewReader(logFile))

	for {
		var entry core.Entry
		err := decoder.Decode(&entry)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		logs = append(logs, entry)
	}

	return logs, nil
}

// func (o *OSFileSystem) Save(writer io.Writer, name string, data []byte, perm fs.FileMode) error {
// 	return writer.Write(o.dir+"/"+name, data, perm)
// }

// func (o *OSFileSystem) Remove(name string) error {
// 	return os.Remove(o.dir + "/" + name)
// }
