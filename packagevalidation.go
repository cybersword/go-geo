package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/cybersword/go-geo/task"
	"github.com/cybersword/go-geo/validation"
)

// 后台服务参数
// var daemon bool
var taskID string
var pathJSON string
var dir string

func init() {
	// flag.BoolVar(&daemon, "d", false, "run as a daemon with -d")
	// flag.BoolVar(&daemon, "daemon", false, "run as a daemon with -d")
	flag.StringVar(&taskID, "t", "91419", "taskid with -t")
	flag.StringVar(&pathJSON, "j", "./url_info.json", "read json in file with -j")
	flag.StringVar(&dir, "w", "./data", "work directory with -w")
	if !flag.Parsed() {
		flag.Parse()
	}

	// if daemon {
	// 	args := os.Args[1:]
	// 	args = append(args, "-d=false")
	// 	cmd := exec.Command(os.Args[0], args...)
	// 	cmd.Start()
	// 	fmt.Println("[PID]", cmd.Process.Pid)
	// 	os.Exit(0)
	// }
}
func main() {
	p := fmt.Println
	// p("是否后台运行:", daemon)
	// dir := flag.Arg(0)
	// pathJSON := flag.Arg(1)
	p("JSON路径:", pathJSON)
	// taskID := "93721"  // 有错误的任务
	// taskID := "97264" // 190 exto
	// taskID := "91419" // 赤峰
	j, err := task.GetJSONByTaskID(taskID)
	// j, err := validation.GetJSONFromFile(pathJSON)
	if err != nil {
		p("faild : ", err)
		return
	}
	pi, err := task.GetPackageInfoByJSON(j)
	if err != nil {
		p("faild : ", err)
		return
	}
	mapURL := map[string]string{}
	for _, mesh := range pi.List {
		meshID := mesh.MeshID
		panos := mesh.ReferURL.ExtoPano
		for _, pano := range panos {
			mapURL[meshID+"_"+pano.PanoID] = pano.URL
		}
	}
	numURL := len(mapURL)
	p("需要检查的文件数目:", numURL)

	chOK := make(chan string, 200)
	chErr := make(chan string, 200)

	for name, url := range mapURL {
		go func(n string, u string) {
			fn := n + ".exto"
			// p(fn, "download start", time.Now())
			validation.Download(u, dir, fn)
			// p(fn, "download end", time.Now())
			exto := validation.InitSqlite3(n, filepath.Join(dir, fn), "exto")
			err := exto.Validate()
			if err != nil {
				chErr <- exto.GetName()
			} else {
				chOK <- exto.GetName()
			}
			// p(fn, "validate end", time.Now())
		}(name, url)
	}

	numErr := 0
	for i := 0; i < numURL; i++ {
		select {
		case <-chErr:
			// p(t, "Error")
			numErr++
		case <-chOK:
			// p(t, "OK")
			// continue
		}
	}
	close(chErr)
	close(chOK)

	p("损坏个数/总数:", numErr, "/", numURL)

}
