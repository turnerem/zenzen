package storage

import (
	"encoding/json"
	"io/fs"

	"github.com/turnerem/zenzen/core"
)

// get all logs from fs
// add log
// delete log

const DIR = "notes"

type FSFileSystem struct {
	dir        string
	fileSystem fs.FS
}

func NewFSFileSystem(dir string, fileSystem fs.FS) *FSFileSystem {
	return &FSFileSystem{dir: dir, fileSystem: fileSystem}
}

func (o *FSFileSystem) GetAll() ([]core.Entry, error) {
	dir, err := fs.ReadDir(o.fileSystem, o.dir)

	if err != nil {
		return nil, err
	}

	var logs []core.Entry
	for _, file := range dir {
		log, err := getLog(o.fileSystem, o.dir+"/"+file.Name())
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func getLog(fileSystem fs.FS, filename string) (core.Entry, error) {
	logFile, err := fs.ReadFile(fileSystem, filename)
	if err != nil {
		return core.Entry{}, err
	}

	var log core.Entry
	err = json.Unmarshal(logFile, &log)
	if err != nil {
		return core.Entry{}, err
	}

	return log, nil
}

// func (o *OSFileSystem) Save(writer io.Writer, name string, data []byte, perm fs.FileMode) error {
// 	return writer.Write(o.dir+"/"+name, data, perm)
// }

// func (o *OSFileSystem) Remove(name string) error {
// 	return os.Remove(o.dir + "/" + name)
// }
