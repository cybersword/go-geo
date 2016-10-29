// Package validation 校验 sqlite 文件是否存在损坏
// 后续可以加上简单的内容校验.
package validation

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

// GKDataType 果壳数据规格, 表结构等.
type GKDataType uint32

// TTFA, TTFB 底图规格
// EXTO 中业 mark, gps.
const (
	UNKNOWN GKDataType = iota
	TTFA
	TTFB
	EXTO
)

// StoreType 存储方式 sqlite3, pg, xml, json 等.
type StoreType uint32

// SQLITE3 sqlite3.
const (
	SQLITE3 StoreType = iota
)

// Validator is the interface indicating the type implementing it supports data validation.
type Validator interface {
	Validate() error
	// 为了返回结果方便
	GetName() string
}

// GKData object.
type GKData struct {
	name       string
	path       string
	storeType  StoreType
	gkDataType GKDataType
}

// Sqlite3 a file struct.
type Sqlite3 struct {
	GKData
	ext string
}

// InitSqlite3 init a Sqlite3 struct
func InitSqlite3(name string, path string, ext string) Sqlite3 {
	switch ext {
	case "exto":
		return Sqlite3{GKData{name, path, SQLITE3, EXTO}, ext}
	case "ttfa":
		return Sqlite3{GKData{name, path, SQLITE3, TTFA}, ext}
	case "ttfb":
		return Sqlite3{GKData{name, path, SQLITE3, TTFB}, ext}
	default:
		return Sqlite3{GKData{name, path, SQLITE3, UNKNOWN}, ext}
	}
}

// Validate implement interface.
func (p *Sqlite3) Validate() error {
	db, err := sql.Open("sqlite3", p.path)
	if err != nil {
		return err
	}
	defer db.Close()
	// 执行 sqlite 命令, 检查文件是否损坏
	rows, err := db.Query("PRAGMA integrity_check;")
	if err != nil {
		return err
	}
	rows.Next()
	var integrityCheck string
	err = rows.Scan(&integrityCheck)
	if err != nil {
		return err
	}
	if integrityCheck != "ok" {
		return errors.New(integrityCheck)
	}
	// 以后可能需要通过 sql 简单检查数据是否符合规格(质检 S 级规则)
	if false {
		rows, err = db.Query("SELECT mark_id, mark_status FROM nav_stv_mark")
		if err != nil {
			return err
		}
		for rows.Next() {
			var markID string
			var markStatus int
			err = rows.Scan(&markID, &markStatus)
			if err != nil {
				return err
			}
			// validate values
			// fmt.Println(markID, markStatus)
		}
	}

	return nil
}

// GetName return name.
func (p *Sqlite3) GetName() string {
	return p.name
}

// Download 后续放到 util 里面.
func Download(url string, dir string, name string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	if err != nil {
		return err
	}
	return nil
}

// GetGKDatas 返回目录中所有指定类型的文件, 暂时直接返回 sqlite3 类型.
func GetGKDatas(dir string, gkDataType GKDataType) []Sqlite3 {
	var ext string
	switch gkDataType {
	case EXTO:
		ext = "exto"
	case TTFA:
		ext = "ttfa"
	default:
		return nil
	}

	matches, err := filepath.Glob(dir + "/*." + ext)
	if err != nil {
		fmt.Println("filepath.Glob() returned ", err)
		return nil
	}
	fmt.Println("glob : ", dir)
	var datas []Sqlite3
	for _, path := range matches {
		datas = append(datas, Sqlite3{GKData{filepath.Base(path), path, SQLITE3, gkDataType}, ext})
	}
	return datas
}

// GetJSONFromFile 后续放到 util 里面.
func GetJSONFromFile(path string) (j string, err error) {
	fi, err := os.Open(path)
	if err != nil {
		return
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)

	// fmt.Println(string(fd))
	j = string(fd)
	return
}
