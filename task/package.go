// Package task 处理曙光任务相关逻辑.
package task

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

// PackageZIP 用于解析 JSON.
type PackageZIP struct {
	Code int
	Msg  string
	Data map[string]string
}

// PackageInfo 对应 package_info_record.url_info 字段的值.
type PackageInfo struct {
	Global map[string]string `json:"global"`
	List   []struct {
		MeshID   string `json:"mesh_id"`
		ReferURL struct {
			ExtoPano []struct {
				PanoID string `json:"pano_id"`
				URL    string `json:"url"`
			} `json:"exto_pano"`
			Qi string `json:"qi"`
		} `json:"refer_url"`
		DataURL string `json:"data_url"`
	} `json:"list"`
}

var dsnDawn string
var dsnMica string

func init() {
	// 设置 MySQL
	// mysql -hgzns-ns-map-guoke46.gzns -uroot -proot guoke_dawn
	dsnDawn = "root:root@tcp(gzns-ns-map-guoke46.gzns.baidu.com:3306)/guoke_dawn?charset=utf8"
	dsnMica = "root:root@tcp(gzns-ns-map-guoke46.gzns.baidu.com:3306)/mica?charset=utf8"
}

// GetPackageInfoByJSON 将 JSON 解析为结构体.
func GetPackageInfoByJSON(j string) (pi PackageInfo, err error) {
	err = json.Unmarshal([]byte(j), &pi)
	return
}

// GetJSONByTaskID 获取任务提交时的任务包信息(JSON字符串).
func GetJSONByTaskID(taskid string) (j string, err error) {
	db, err := sql.Open("mysql", dsnDawn)
	if err != nil {
		return
	}
	defer db.Close()
	// 查询提交的任务包
	s := "SELECT url_info FROM task, package_info_record WHERE up_pir_id=pir_id AND task.task_id="
	rows, err := db.Query(s + taskid)
	if err != nil {
		return
	}
	if !rows.Next() {
		err = errors.New("Commit task package info not exists : " + taskid)
		return
	}
	err = rows.Scan(&j)

	return
}

// GetPackageZIP 返回打包过的数据的下载地址.
func GetPackageZIP(taskid string, dir string) (url string, err error) {
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
	var pzip PackageZIP
	err = json.Unmarshal(htmlData, &pzip)
	if err != nil {
		return
	}
	code := pzip.Code
	if code != 0 {
		err = errors.New(pzip.Msg)
		return
	}
	url = pzip.Data["internal_url"]

	return
}
