package main

import (
	"svn-deploy-go/lib"
	"flag"
	"runtime"
	"log"	
	"os/exec"
	"bytes"
	"strconv"
	"time"
	"os"
	"strings"
	"bufio"
	"path"
)

func getSvnPath() string{
	cmd := exec.Command("where", "svn")

	var outbuffer bytes.Buffer
	var Stderr bytes.Buffer
	cmd.Stderr = &Stderr
	cmd.Stdout = &outbuffer
	err := cmd.Run()
	if err != nil {
		log.Println(err)
		log.Println(Stderr.String())
		return ""
	}

	svnbins := outbuffer.String()
	s := strings.Split(svnbins, "\n")
	if len(s) > 1 {
		svnbin := s[len(s)-2]
		svnbin = strings.Replace(svnbin, "\r", "", -1)
		svnbin = strings.Replace(svnbin, "\n", "", -1)
		return svnbin
	}else{
		svnbin := s[len(s)-1]
		svnbin = strings.Replace(svnbin, "\r", "", -1)
		svnbin = strings.Replace(svnbin, "\n", "", -1)
		return svnbin
	}
}

func _init() {
	t := time.Now().Unix()

	file := "./version/" +"version"+ strconv.FormatInt(t,10) +".txt"
	dist_dir := path.Dir(file)
	if _, _err := os.Stat(dist_dir); _err != nil && os.IsNotExist(_err) {
		os.MkdirAll(dist_dir, 0777)
	}

	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile) // 将文件设置为log输出的文件
	log.SetPrefix("[log]")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	return
}


func main() {
	log.Println("Start Svn Sync")
	_init()
	svnbin := getSvnPath()
	log.Println(svnbin)

	conf_path := flag.String("c", "config.json", "config json file")	
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())
	
	err, c := lib.NewConfig(*conf_path)
	if err != nil {
		return
	}
	
	for idx, x := range c.Persons {
		svn_item := &lib.SVNItem{Name:x.Name,URL:x.URL,Username:x.Username,Password:x.Password, LocalPath:x.LocalPath}
		worker := lib.NewSVNWrapper(svnbin, svn_item)
		_,_, err, new_version := worker.PackageUpdate(strconv.Itoa(x.Lastver), "HEAD",  x.LocalPath)
		if err != nil{
			log.SetOutput(os.Stderr)
			log.Println("Err return ........................................................................... !Press Any key to continue!!! ")

			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
			return
		}
		c.Persons[idx].Lastver,_ = strconv.Atoi(new_version)
		log.Println(x.Lastver)
	}
	
	c.Save()

	log.Println("Succ @_@ ")
	log.SetOutput(os.Stderr)
	log.Println(" Press Any key to continue!!! Succ @_@")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}
