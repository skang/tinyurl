package store

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/tinyurl/tinyurl/entity"
)

// GeneralDBClient support sqlite3, mysql and so on...
type GeneralDBClient struct {
	GormDB     *gorm.DB
	DBName     string
	DBType     string
	configPath string
}

// NewGeneralDBClient
func NewGeneralDBClient(configPath string) *GeneralDBClient {
	setting := entity.GetGlobalConfig(configPath)
	switch setting.DBType {
	case entity.SQLITE3:
		// sqlite3 does not need InitDB
		logrus.Infof("database is %s, InitDB will done when Open connection.\n", entity.SQLITE3)
	case entity.MYSQL:
		InitDB(setting)
	default:
		InitDB(setting)
	}

	source := getDBSourceWithDatabase(setting)
	gclient := &GeneralDBClient{}
	// should create db first before open connection.
	db, err := gorm.Open(setting.DBType, source)
	if err != nil {
		logrus.Fatalf("open connection to mysql use gorm error: %s", err)
	}

	gclient.GormDB = db
	gclient.DBName = setting.DBName
	gclient.DBType = setting.DBType
	gclient.configPath = configPath
	gclient.GormDB.AutoMigrate(&entity.URL{})
	gclient.GormDB.AutoMigrate(&entity.SenderWorker{})
	gclient.GormDB.FirstOrCreate(&entity.SenderWorker{}, entity.SenderWorker{ID: 1, Index: 0})

	return gclient
}

func getDBSourceWithDatabase(setting *entity.GlobalConfig) string {
	source := ""
	switch setting.DBType {
	case entity.SQLITE3:
		source = fmt.Sprintf("%s/%s", setting.DBPath, setting.DBName)
	case entity.MYSQL:
		source = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
			setting.DBUser, setting.DBPassword, setting.DBHost, setting.DBPort, setting.DBName)
	default:
		source = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
			setting.DBUser, setting.DBPassword, setting.DBHost, setting.DBPort, setting.DBName)
	}

	return source
}

func getDBSource(setting *entity.GlobalConfig) string {
	source := ""
	switch setting.DBType {
	case entity.SQLITE3:
		source = fmt.Sprintf("%s/%s", setting.DBPath, setting.DBName)
		break
	case entity.MYSQL:
		source = fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8&parseTime=True&loc=Local",
			setting.DBUser, setting.DBPassword, setting.DBHost, setting.DBPort)
		break
	default:
		source = fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8&parseTime=True&loc=Local",
			setting.DBUser, setting.DBPassword, setting.DBHost, setting.DBPort)
	}

	return source
}

// CRUD for URL

func (gclient *GeneralDBClient) Create(url *entity.URL) {
	gclient.GormDB.Create(url)
}

func (gclient *GeneralDBClient) Update(url *entity.URL) {
	gclient.GormDB.Save(url)
}

func (gclient *GeneralDBClient) GetByOriginURL(originURL string) *entity.URL {
	var url entity.URL
	gclient.GormDB.Where("origin_url = ?", originURL).First(&url)

	return &url
}

func (gclient *GeneralDBClient) GetByShortPath(shortPath string) *entity.URL {
	var url entity.URL
	gclient.GormDB.Where("short_path = ?", shortPath).First(&url)

	return &url
}

// CRUD for SenderWorker

func (gclient *GeneralDBClient) UpdateSenderWorker(sender *entity.SenderWorker) {
	// gclient.GormDB.Update(sender)
	gclient.GormDB.Model(sender).Update("index", sender.Index)
}

func (gclient *GeneralDBClient) SaveSenderWorker(sender *entity.SenderWorker) {
	gclient.GormDB.Save(sender)
}

func (gclient *GeneralDBClient) GetSenderWorker() *entity.SenderWorker {
	sender := entity.SenderWorker{}
	gclient.GormDB.First(&sender)

	return &sender
}

// DropDatabase drop self hold database
func (gclient *GeneralDBClient) DropDatabase() {
	switch gclient.DBType {
	case entity.SQLITE3:
		// sqlite does not have DROP DATABASE command, we just delete file
		setting := entity.GetGlobalConfig(gclient.configPath)
		source := fmt.Sprintf("%s/%s", setting.DBPath, setting.DBName)
		if err := os.Remove(source); err != nil {
			logrus.Fatalf("drop database %s error: %v", gclient.DBName, err)
		}
	case entity.MYSQL:
		sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s;", gclient.DBName)
		db := gclient.GormDB.DB()

		_, err := db.Exec(sql)
		if err != nil {
			logrus.Fatalf("drop database %s error: %v", gclient.DBName, err)
		}
	}
}

// InitDB doing init work of db:create database...
func InitDB(setting *entity.GlobalConfig) {
	source := ""
	switch setting.DBType {
	case entity.SQLITE3:
		logrus.Info("InitDB has done when new client, skip.")
		return
	case entity.MYSQL:
		source = fmt.Sprintf("%s:%s@tcp(%s:%s)/",
			setting.DBUser, setting.DBPassword, setting.DBHost, setting.DBPort)
	default:
		source = fmt.Sprintf("%s:%s@tcp(%s:%s)/",
			setting.DBUser, setting.DBPassword, setting.DBHost, setting.DBPort)
	}

	db, err := sql.Open(setting.DBType, source)
	if err != nil {
		logrus.Fatalf("connection to db error: %s", err)
	}
	defer db.Close()

	sql := "CREATE DATABASE IF NOT EXISTS " + setting.DBName + ";"
	_, err = db.Exec(sql)
	if err != nil {
		logrus.Fatalf("create db %s error: %v", setting.DBName, err)
	}
}
