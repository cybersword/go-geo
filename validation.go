package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

// TaskPackageInfo 解析 JSON 用
type TaskPackageInfo struct {
	Code int
	Msg  string
	Data map[string]string
}

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

//
func download(url string, dir string, name string) error {
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

func getTaskPackageInfoFromMySQL(taskid string) (tpi [2]TaskPackageInfo, err error) {
	// mysql -hgzns-ns-map-guoke46.gzns -uroot -proot guoke_dawn
	db, err := sql.Open("mysql", "root:root@tcp(gzns-ns-map-guoke46.gzns.baidu.com:3306)/guoke_dawn?charset=utf8")
	if err != nil {
		return
	}
	rows, err := db.Query("SELECT down_pir_id, up_pir_id FROM task WHERE task_id=" + taskid)
	if err != nil {
		return
	}
	if !rows.Next() {
		err = errors.New("task_id not exists: " + taskid)
		return
	}
	var downloadID string
	var uploadID string
	err = rows.Scan(&downloadID, &uploadID) // 131595, 131892
	if err != nil {
		return
	}
	fmt.Println(downloadID, uploadID)

	return
}

func getTaskPackage(taskid string, dir string) (path string, err error) {
	urlQuery := "http://guoke.map.baidu.com/api/mica/api/fix?action=getdawntask&task_id=" + taskid

	resp, err := http.Get(urlQuery)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	htmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	//fmt.Println(string(htmlData))
	var info TaskPackageInfo
	err = json.Unmarshal(htmlData, &info)
	if err != nil {
		return
	}
	code := info.Code
	if code != 0 {
		err = errors.New(info.Msg)
		return
	}
	urlTask := info.Data["internal_url"]
	name := "task_pkg_" + taskid + ".zip"
	path = filepath.Join(dir, taskid)
	err = os.Mkdir(path, 0777)
	if err != nil {
		return
	}
	err = download(urlTask, path, name)
	if err != nil {
		return
	}

	return
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
	numErr := 0
	for range extoList {
		select {
		case t := <-chErr:
			fmt.Println(t, "Error")
			numErr++
		case t := <-chOK:
			fmt.Println(t, "OK")
			continue
		}
	}
	close(chErr)
	close(chOK)
	fmt.Printf("err num : %v / %v \n", numErr, len(extoList))

	taskID := "93721"
	_, err := getTaskPackageInfoFromMySQL(taskID)

	//path, err := getTaskPackage(taskID, "/Users/baidu/work")
	if err != nil {
		fmt.Println("download faild : ", err)
	}
	//fmt.Println("download : ", path)
}
