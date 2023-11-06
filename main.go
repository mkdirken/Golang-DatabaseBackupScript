package main

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jlaffaye/ftp"
)

// Database Info
const dbUser = "root"
const dbPass = "1234"

var dbNames = []string{"table1", "table2"}

const dbHost = "127.0.0.1" // MySQL sunucunuzun IP adresi veya hostname
const dbPort = "3306"      // MySQL sunucunuzun port numarası

// BACKUP FTP SERVER
const ftpServerIp = "192.168.1.104"
const ftpServerPort = "21"
const ftpServerUsername = "root"
const ftpServerPassword = "asdasd"
const ftpServerFolder = "/DatabaseBackup"

var backupFolder = "backup"
var backupFilePrefix = "databaseBackup_"

func main() {

	currentDirectory, err := os.Getwd()
	if err != nil {
		fmt.Println("Dizin alınamadı:", err)
		return
	}
	backupFolder = currentDirectory + "/" + backupFolder
	backupFolderControl(backupFolder) // Backup Folder Control
	for {
		go backupMysqlMariadb()
		time.Sleep(time.Second * 10)
	}

}

func backupMysqlMariadb() {
	// MySQL/MariaDB bağlantısı kurun
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbNames[0]))
	if err != nil {
		fmt.Println("Veritabanı bağlantısı kurulamadı:", err)
		return
	}
	defer db.Close()

	for _, dbName := range dbNames {
		backupFilePrefix := backupFilePrefix + dbName + "_"
		// Backup File
		backupFileName := fmt.Sprintf(backupFilePrefix+"%s.sql", time.Now().Format("20060102_150405"))
		backupFilePath := backupFolder + "/" + backupFileName

		// MySQL/MariaDB veritabanını dışa aktar
		cmd := exec.Command("mysqldump", fmt.Sprintf("-u%s", dbUser), fmt.Sprintf("-p%s", dbPass), dbName, "--result-file="+backupFilePath)
		err = cmd.Run()
		if err != nil {
			fmt.Println("Yedekleme işlemi başarısız oldu:", err)
			return
		}

		fmt.Printf("Yedekleme başarıyla oluşturuldu: %s\n", backupFileName)

		sqlZip(backupFileName)

		uploadFTP(backupFilePrefix, backupFileName, backupFilePath)

		// Remove File
		os.Remove(backupFilePath)
		os.Remove(backupFilePath + ".zip")
	}

}

func uploadFTP(backupFilePrefix, backupFileName, backupFilePath string) {
	// FTP sunucusuna bağlan
	conn, err := ftp.Connect(ftpServerIp + ":" + ftpServerPort)
	if err != nil {
		fmt.Println("FTP sunucusuna bağlanma hatası:", err)
		return
	}
	defer conn.Quit()

	// Kullanıcı adı ve şifre ile oturum aç
	err = conn.Login(ftpServerUsername, ftpServerPassword)
	if err != nil {
		fmt.Println("Oturum açma hatası:", err)
		return
	}

	// Dosya yükleme işlemi
	file, err := os.Open(backupFilePath + ".zip")
	if err != nil {
		fmt.Println("Dosya açma hatası:", err)
		return
	}
	defer file.Close()

	err = conn.Stor(ftpServerFolder+"/"+backupFileName+".zip", file)
	if err != nil {
		fmt.Println("Dosya yükleme hatası:", err)
		return
	}
	// Klasörü boşaltma işlemi
	klasorIci, err := conn.List(ftpServerFolder + "/")
	if err != nil {
		fmt.Println("Klasör içi listeleme hatası:", err)
		return
	}

	for _, dosya := range klasorIci {

		if strings.HasPrefix(dosya.Name, backupFilePrefix) && backupFileName+".zip" != dosya.Name {
			fmt.Println("Dosya Silindi : " + dosya.Name)
			conn.Delete(ftpServerFolder + "/" + dosya.Name)
		}
	}

	fmt.Println("Dosya başarıyla yüklendi.")

}

func sqlZip(backupFileName string) {

	// Yeni bir zip dosyası oluşturun
	zipFile, err := os.Create(backupFolder + "/" + backupFileName + ".zip")
	if err != nil {
		fmt.Println("Zip dosyası oluşturulamadı:", err)
		return
	}
	defer zipFile.Close()

	// Zip yaratıcıyı oluşturun
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Dosyayı açın
	fileToZip, err := os.Open(backupFolder + "/" + backupFileName)
	if err != nil {
		fmt.Println("Dosya açılamadı:", err)
		return
	}
	defer fileToZip.Close()

	// Zip dosyasına dosyayı ekle
	fileInZip, err := zipWriter.Create(backupFileName)
	if err != nil {
		fmt.Println("Zip dosyasına dosya eklenemedi:", err)
		return
	}

	_, err = io.Copy(fileInZip, fileToZip)
	if err != nil {
		fmt.Println("Dosya kopyalanamadı:", err)
		return
	}

	fmt.Println("Dosya başarıyla sıkıştırıldı:", backupFileName+".zip")

}

func backupFolderControl(dirPath string) {
	// Dizinin varlığını kontrol edin
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		fmt.Printf("Dizin %s bulunamadı. Oluşturuluyor...\n", dirPath)
		// Dizin yoksa oluşturun
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			fmt.Println("Dizin oluşturulamadı:", err)
		} else {
			fmt.Printf("Dizin %s başarıyla oluşturuldu.\n", dirPath)
		}
	} else {
		fmt.Printf("Dizin %s mevcut.\n", dirPath)
	}
}
