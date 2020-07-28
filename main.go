package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

var path = "c:\\storage\\"

type fileInfo struct {
	FileName string `json:"file_name"`
	FleSize  int64  `json:"fle_size"`
}

type snapshot struct {
	DirName string     `json:"dir_name"`
	Size    int64      `json:"size"`
	Files   []fileInfo `json:"files"`
	Date    time.Time  `json:"date"`
}

func getRootDirectories(path string) []string {
	var dirList []string

	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("[getRootDirectories][Ошибка чтения корневого каталога] ", err)
	}

	for _, dir := range files {
		if dir.IsDir() == true {
			dirList = append(dirList, dir.Name())
		}
	}

	return dirList
}

func getFileListFromDir(dirName string) snapshot {

	fileList, err := ioutil.ReadDir(path + dirName)
	if err != nil {
		fmt.Println("[getFileListFromDir][Ошибка получения списка файлов из директории][", dirName, "]", err)
	}

	var dirSize int64
	var dirSnapshot snapshot

	dirSnapshot.DirName = dirName
	for _, fileName := range fileList {
		var currentFile fileInfo
		currentFile.FileName = fileName.Name()
		currentFile.FleSize = fileName.Size()

		dirSnapshot.Files = append(dirSnapshot.Files, currentFile)
		dirSize = dirSize + fileName.Size()
	}

	dirSnapshot.Size = dirSize
	dirSnapshot.Date = time.Now()
	return dirSnapshot
}

func main() {
	fmt.Println("run")

	dirList := getRootDirectories(path)

	var snapshotInfo []snapshot

	for _, currentDir := range dirList {
		snaps := getFileListFromDir(currentDir)
		snapshotInfo = append(snapshotInfo, snaps)
	}

	stor, err := json.Marshal(snapshotInfo)
	if err != nil {
		fmt.Println("Ошибка преобразования структуры снепшота в json", err)
	}

	out := string(stor)

	fmt.Println("current json = ", out)

	err = ioutil.WriteFile("c:\\storage\\info.txt", stor, 0777)
	if err != nil {
		fmt.Println("Ошибка записи файла", err)
	}

	//fmt.Println(getRootDirectories(path))
}
