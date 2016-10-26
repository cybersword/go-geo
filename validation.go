package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// GKDataType 果壳数据规格, 表结构等
type GKDataType uint32

// TTFA, TTFB 底图规格
// EXTO 中业 mark, gps
const (
	TTFA GKDataType = iota
	TTFB
	EXTO
)

// StoreType 存储方式 sqlite3, pg, xml, json 等
type StoreType uint32

// SQLITE3 sqlite3
const (
	SQLITE3 StoreType = iota
)

// Validator is the interface indicating the type implementing it supports data validation.
type Validator interface {
	Validate() error
	// 为了返回结果方便
	GetName() string
}

// GKData object
type GKData struct {
	name       string
	path       string
	storeType  StoreType
	gkDataType GKDataType
}

// Sqlite3 a file struct
type Sqlite3 struct {
	GKData
	ext string
}

// Validate implement interface
func (p *Sqlite3) Validate() error {
	db, err := sql.Open("sqlite3", p.path)
	if err != nil {
		//fmt.Println(f.path, "Error: ", err)
		return err
	}
	defer db.Close()

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
	//fmt.Println(integrityCheck)

	// validate table
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
		// fmt.Println(markID, markStatus)
	}

	return nil
}

// GetName return name
func (p *Sqlite3) GetName() string {
	return p.name
}

// 后台服务参数
var daemon bool

func init() {
	flag.BoolVar(&daemon, "d", false, "run as a daemon with -d")
	flag.BoolVar(&daemon, "daemon", false, "run as a daemon with -d")
	if !flag.Parsed() {
		flag.Parse()
	}

	if daemon {
		args := os.Args[1:]
		args = append(args, "-d=false")
		cmd := exec.Command(os.Args[0], args...)
		cmd.Start()
		fmt.Println("[PID]", cmd.Process.Pid)
		os.Exit(0)
	}
}

// 返回目录中所有指定类型的文件, 暂时直接返回 sqlite3 类型
func getGKDatas(dir string, gkDataType GKDataType) []Sqlite3 {
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

func main() {
	fmt.Println("daemon :", daemon)
	dir := flag.Arg(0)
	chOK := make(chan string, 5)
	chErr := make(chan string, 5)
	extoList := getGKDatas(dir, EXTO)
	for _, exto := range extoList {
		go func(t Validator) {
			e := t.Validate()
			if e != nil {
				chErr <- t.GetName()
			} else {
				chOK <- t.GetName()
			}
		}(&exto)
	}
	errNUm := 0
	for range extoList {
		select {
		case t := <-chErr:
			fmt.Println(t, "Error")
			errNUm++
		case t := <-chOK:
			fmt.Println(t, "OK")
			continue
		}
	}
	close(chErr)
	close(chOK)
	fmt.Printf("err num : %v / %v \n", errNUm, len(extoList))
}
