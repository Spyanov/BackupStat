package main

import (
	"fmt"
	"io/ioutil"
)

type snapshot struct {
	Name string `json:"name"`
	Size int64  `json:"size"`

	Files []string `json:"files"`
}

func getRootDirectories(path string) []snapshot {

	var dirList []snapshot

	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("[getRootDirectories][Ошибка чтения корневого каталога] ", err)
	}
	for _, dir := range files {
		if dir.IsDir() == true {
			var currentDirInfo snapshot
			currentDirInfo.Name = dir.Name()

			currentDirInfo.Size = dir.Size()

			dirList = append(dirList, currentDirInfo)
		}
	}
	return dirList
}

func main() {
	fmt.Println("run")

	var path = "c:\\storage"
	//getRootDirectories(path)

	fmt.Println(getRootDirectories(path))
}
