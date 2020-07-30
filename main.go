package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/martini-contrib/cors"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var path = "c:\\storage"

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
	Hash    string     `json:"hash"`
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

	fileList, err := ioutil.ReadDir(path + "\\" + dirName)
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
		fmt.Println("[createClientsOnTheServer][sql.open]", err)
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

	//проверяем есть ли имя локальной папки в БД
	for _, dir := range localDirList {
		if contains(clientList, dir) == true {
			//есть в бд, перескакиваем
			continue
		} else {
			//нет в БД, создаем пользователя с именем локальной папки
			qs := "INSERT INTO Clients(Name,Folder) VALUES('" + dir + "','" + dir + "');"
			_, err := db.Exec(qs)
			if err != nil {
				fmt.Println("Ошибка добавления клиента", err)
			}
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
	hasher := md5.New()
	hasher.Write(hash)
	return hex.EncodeToString(hasher.Sum(nil))
}

func writeSnapshot(record snapshot) {
	db, err := sql.Open("mysql", "backupService:NYwU8t2yHtERcMnU!*@tcp(backup.xkc1.ru:3306)/backupLog")
	if err != nil {
		fmt.Println("[writeSnapshot][sql.open]", err)
	}
	defer db.Close()

	hash := GenerateToken(time.Now())

	_, err = db.Exec("INSERT INTO snapshots (Name,Date, Size, Hash) VALUES (?,?,?,?)", record.DirName, record.Date, record.Size, hash)
	if err != nil {
		fmt.Println("[writeSnapshot][db.Exec=INSERT INTO snapshots (Name,Date, Size, Hash)]", err)
	}
	if err == nil {
		log.Println("[writeSnapshot][", record.DirName, "][Добавлено задание:", hash, "]")
	}

	for _, file := range record.Files {
		_, err := db.Exec("INSERT INTO files (Hash, File, Size) VALUES (?,?,?)", hash, file.FileName, file.FleSize)
		if err != nil {
			fmt.Println("[writeSnapshot][db.Exec=INSERT INTO files (Hash, File, Size)]", err)
		}
		if err == nil {
			log.Println("	[writeSnapshot][Добавлен файл", file.FileName, " к заданию :", hash, "]")
		}
	}

}

func index(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "backupService:NYwU8t2yHtERcMnU!*@tcp(backup.xkc1.ru:3306)/backupLog")
	if err != nil {
		fmt.Println("[writeSnapshot][sql.open]", err)
	}
	defer db.Close()

	var snapshots []snapshot

	rows, err := db.Query("SELECT `Name`, `Size`, `Hash`  FROM `snapshots` WHERE `Name`='customer1'")
	for rows.Next() {
		var currentSnap snapshot
		err := rows.Scan(&currentSnap.DirName, &currentSnap.Size, &currentSnap.Hash)
		if err != nil {
			fmt.Println("[index][Ошибка получения данных из таблицы snapshots]", err)
		}

		snapshots = append(snapshots, currentSnap)
	}

	fmt.Println("snp=", snapshots)

	result, err := json.Marshal(snapshots)
	if err != nil {
		fmt.Println("Ошибка преобразования в json")
	}

	w.Write(result)

}

func play() {
	//Строим список корневых директорий с файлами
	localDirList := getRootDirectories(path)
	//Создаем на сервере нового клиента если появилась новая папка в корневом каталоге
	// имя нового клиента соотвествует имени новой папки
	createClientsOnTheServer(localDirList)

	for _, oldFolder := range localDirList {
		curSnapshot := createSnapshot(oldFolder)
		writeSnapshot(curSnapshot)
	}
}

func main() {
	fmt.Println("run")

	m := martini.Classic()
	m.Use(cors.Allow(&cors.Options{
		AllowOrigins:     []string{"http://localhost"},
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "PATCH"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	m.Get("/", index)
	m.Get("/play", play)

	m.RunOnAddr(":4000")
}
