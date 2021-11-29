package lib

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"log"
)

type LogEntry struct {
	Version int    `xml:"revision,attr"`
	Author  string `xml:"author"`
	Date    string `xml:"date"`
	Msg     string `xml:"msg"`
}

type Log struct {
	XMLName  xml.Name   `xml:"log"`
	LogEntry []LogEntry `xml:"logentry"`
}

type DiffEntry struct {
	Kind string `xml:"kind,attr"`
	Type string `xml:"item,attr"`
	Path string `xml:",chardata"`
}

type Diff struct {
	XMLName xml.Name    `xml:"diff"`
	Paths   []DiffEntry `xml:"paths>path"`
}

type SVNInfo struct {
	XMLName xml.Name     `xml:"info"`
	Entry   SVNInfoEntry `xml:"entry"`
}

type SVNInfoEntry struct {
	Kind           string `xml:"kind,attr"`
	Path           string `xml:"path,attr"`
	LastVersion    string `xml:"revision,attr"`
	URL            string `xml:"url"`
	ROOT           string `xml:"repository>root"`
	LastAuthor     string `xml:"commit>author"`
	LastChangeDate string `xml:"commit>date"`
}

type SVNItem struct {
	Name      string  `json:"name"`
	URL       string  `json:"url"`
	Username  string  `json:"username"`
	Password  string  `json:"password"`
	LocalPath string  `json:"local"`
}

type SVNWrapper struct {
	config   *SVNItem
	bin_path string
}

func NewSVNWrapper(SVNBin string, svn_item *SVNItem) *SVNWrapper {
	return &SVNWrapper{config: svn_item, bin_path: SVNBin}
}

func (s *SVNWrapper) Run(cmd *exec.Cmd, stdout *bytes.Buffer) (err error) {
	var Stderr bytes.Buffer
	cmd.Stderr = &Stderr
	cmd.Stdout = stdout
	log.Println("start run local cmd:", cmd)
	err = cmd.Run()
	if err != nil {
		log.Println(err)
		log.Println(Stderr.String())
		return
	}
	return
}

func (s *SVNWrapper) Build_SVN_CMD(arg ...string) (cmd *exec.Cmd) {
	default_cmd := []string{"--username", s.config.Username, "--password", s.config.Password}
	args := append(default_cmd, arg...)
	cmd = exec.Command(s.bin_path, args...)
	return
}

func (s *SVNWrapper) ShowDiff(start string, end string) (out *Diff, err error) {
	cmd := s.Build_SVN_CMD("diff", s.config.URL, "-r", fmt.Sprintf("%s:%s", start, end), "--summarize", "--xml")
	var outbuffer bytes.Buffer
	err = s.Run(cmd, &outbuffer)
	if err != nil {
		return
	}
	out = &Diff{}
	err = xml.Unmarshal(outbuffer.Bytes(), out)
	if err != nil {
		log.Println("Unmarshal error:", err)
	}
	return
}

func (s *SVNWrapper) Export(export_path string, version string, save_path string) (err error) {
	dist_path := path.Join(save_path, strings.Replace(export_path, s.config.URL, "", 1))
	dist_dir := path.Dir(dist_path)
	if _, _err := os.Stat(dist_dir); _err != nil && os.IsNotExist(_err) {
		os.MkdirAll(dist_dir, 0777)
	}
	cmd := s.Build_SVN_CMD("export", "-r", version, export_path, dist_path, "--force")
	var outbuffer bytes.Buffer
	err = s.Run(cmd, &outbuffer)
	if err != nil {
		return
	}	
	return
}

func (s *SVNWrapper) Exports(export_paths []string, version string, save_path string) (logs []string, zipfile_path string) {
	logs = make([]string, 0)

	for _, export_path := range export_paths {
		export_path = strings.TrimSpace(export_path)
		if len(export_path) == 0 {
			continue
		}
		err := s.Export(export_path, version, s.config.LocalPath)
		if err != nil {
			logs = append(logs, fmt.Sprintf("%s", err))
		}
		logs = append(logs, "")
	}

	// if len(export_paths) > 0 {
	// 	version_file := save_path
	// 	err1 := ioutil.WriteFile(version_file, []byte(version), 0777)
	// 	if err1 != nil {
	// 		log.Println(err1)
	// 	}
	// }

	//zipfile_path = fmt.Sprintf("%s/%d.zip", save_path, time.Now().UnixNano())
	// ZipFolder(local_path, zipfile_path)
	// //remove local export temp folder
	// err2 := os.RemoveAll(local_path)
	// if err2 != nil {
	// 	log.Println(err2)
	// }
	return
}

func ZipFolder(save_path string, filename string) (err error) {

	zipfile, e := os.Create(filename)
	if e != nil {
		log.Println("create file error:", e)
		return e
	}
	defer zipfile.Close()

	zipWriter := zip.NewWriter(zipfile)
	defer zipWriter.Close()

	save_path = filepath.FromSlash(save_path)

	err = filepath.Walk(save_path, func(_path string, info os.FileInfo, err error) (_e error) {
		if err != nil {
			log.Println("Walk file error:", err)
			return
		}
		if info.Mode().IsDir() {
			return
		}
		f, _err := os.Open(_path)
		if _err != nil {
			log.Println("open file error:", _err)
			return _err
		}
		defer f.Close()

		h := new(zip.FileHeader)
		h.Name = filepath.ToSlash(strings.TrimLeft(strings.Replace(_path, save_path, "", -1), "\\"))
		h.Method = zip.Store
		h.SetModTime(info.ModTime().UTC())

		w, __e := zipWriter.CreateHeader(h)
		if __e != nil {
			log.Println(__e)
			return __e
		}
		if _, ___e := io.Copy(w, f); ___e != nil {
			log.Println(___e)
			return ___e
		}

		return
	})

	return
}

func (s *SVNWrapper) GetLastInfo() (info *SVNInfo, err error) {
	cmd := s.Build_SVN_CMD("info", "-r", "HEAD", s.config.URL, "--xml")
	var outbuffer bytes.Buffer
	err = s.Run(cmd, &outbuffer)
	if err != nil {
		log.Println(err)
		return
	}
	info = &SVNInfo{}
	err = xml.Unmarshal(outbuffer.Bytes(), info)
	if err != nil {
		log.Println("Unmarshal error:", err)
	}

	return
}

func (s *SVNWrapper) PackageUpdate(last_version string, current_version string, save_path string) (pkgFile string, deleteFiles []string, out_err error, new_version string) {
	if current_version == "HEAD" {
		info, err := s.GetLastInfo()
		if err == nil {
			current_version = info.Entry.LastVersion
		}
	}

	if last_version == "0" {
		_, pkgFile = s.Exports([]string{s.config.URL}, current_version, save_path)
		deleteFiles = []string{}
		new_version = current_version
		return
	}

	diff, err := s.ShowDiff(last_version, current_version)
	if err != nil {
		out_err = err
		return
	}
	paths := []string{}
	delete_paths := []string{}
	for _, item := range diff.Paths {
		if item.Kind == "file" && item.Type != "deleted" {
			paths = append(paths, item.Path)
		}
		if item.Type == "deleted" {
			delete_paths = append(delete_paths, strings.Replace(item.Path, s.config.URL, "", 1))
		}
	}

	_, zipfile := s.Exports(paths, current_version, save_path)
	pkgFile = zipfile
	deleteFiles = delete_paths
	new_version = current_version
	log.Println(new_version)
	return
}
