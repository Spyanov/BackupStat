package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"

	"os"
	"path/filepath"
	"time"
)

var path = "."

type clients struct {
	Id         int    `json:"id"`
	ClientName string `json:"client_name"`
	FolderName string `json:"folder_name"`
}

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
		if (dir.Name() == ".git") || (dir.Name() == ".idea") {
			continue
		}
		if dir.IsDir() == true {
			dirList = append(dirList, dir.Name())
		}
	}

	return dirList
}

func createSnapshot(dirName string) snapshot {

	/* работа в текущей директории */
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)
	/* работа в текущей директории */

	fileList, err := ioutil.ReadDir(dir + "\\" + dirName)
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

func contains(clientList []clients, localDir string) bool {
	/*
		Проверяем есть ли такой клиент уже на сервере
	*/
	for _, n := range clientList {
		if localDir == n.FolderName {
			return true
		}
	}
	return false
}

func createClientsOnTheServer(localDirList []string) {
	db, err := sql.Open("mysql", "backupService:NYwU8t2yHtERcMnU!*@tcp(backup.xkc1.ru:3306)/backupLog")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("select * from Clients")
	if err != nil {
		fmt.Println("Ошибка выполнения запроса", err)
	}
	defer rows.Close()

	var clientList []clients
	for rows.Next() {
		var currentClient clients
		err := rows.Scan(&currentClient.Id, &currentClient.ClientName, &currentClient.FolderName)
		if err != nil {
			fmt.Println("Ошибка разбора SQL строки", err)
		}
		clientList = append(clientList, currentClient)
	}

	for _, dir := range localDirList {
		fmt.Println("current dir = ", dir)
		if contains(clientList, dir) == true {
			fmt.Println(dir, " есть на сервере")
			continue
		} else {
			fmt.Println(dir, "нет на сервере")
			qs := "INSERT INTO Clients(Name,Folder) VALUES('" + dir + "','" + dir + "');"
			fmt.Println("qs=", qs)
			_, err := db.Exec(qs)
			if err != nil {
				fmt.Println("Ошибка добавления клиента", err)
			}
			fmt.Println("Добавлен клиент: ", dir)

			currentSnapshot := createSnapshot(dir)
			writeSnapshot(currentSnapshot)
		}

	}
}

func GenerateToken(t time.Time) string {

	z := t.Format("20060102150405")

	hash, err := bcrypt.GenerateFromPassword([]byte(z), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Hash to store:", string(hash))

	hasher := md5.New()
	hasher.Write(hash)
	return hex.EncodeToString(hasher.Sum(nil))
}

func writeSnapshot(record snapshot) {
	db, err := sql.Open("mysql", "backupService:NYwU8t2yHtERcMnU!*@tcp(backup.xkc1.ru:3306)/backupLog")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	hash := GenerateToken(time.Now())

	result, err := db.Exec("INSERT INTO snapshots (Name,Date, Size, Hash) VALUES (?,?,?,?)", record.DirName, record.Date, record.Size, hash)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.RowsAffected()) // id добавленного объекта

	for _, file := range record.Files {
		_, err := db.Exec("INSERT INTO files (Hash, File, Size) VALUES (?,?,?)", hash, file.FileName, file.FleSize)
		if err != nil {
			panic(err)
		}
	}

}

func main() {
	fmt.Println("run")

	//Строим список корневых директорий с файлами
	localDirList := getRootDirectories(path)
	//Создаем на сервере нового клиента если появилась новая папка в корневом каталоге
	// имя нового клиента соотвествует имени новой папки
	createClientsOnTheServer(localDirList)

	for _, oldFolder := range localDirList {
		curSnapshot := createSnapshot(oldFolder)
		writeSnapshot(curSnapshot)
	}

	//var snapshotInfo []snapshot

	//Обходим каждую директорию получая информацию о ее размере, именах файлах и их размере
	//помещаем результат в срез snapshotInfo.
	//for _, currentDir := range dirList {
	//	snaps := getFileListFromDir(currentDir)
	//	snapshotInfo = append(snapshotInfo, snaps)
	//
	//}

	//Пишем в бд информацию о текущем snapshot

	//fmt.Println("Локальные каталоги", localDirList)
	//fmt.Println("Клиенты на сервере", clientList)

	//
	//stor, err := json.Marshal(snapshotInfo)
	//if err != nil {
	//	fmt.Println("Ошибка преобразования структуры снепшота в json", err)
	//}
	//
	//out := string(stor)
	//
	//fmt.Println("current json = ", out)
	//
	//err = ioutil.WriteFile("c:\\storage\\info.txt", stor, 0777)
	//if err != nil {
	//	fmt.Println("Ошибка записи файла", err)
	//}

	//fmt.Println(getRootDirectories(path))
}
