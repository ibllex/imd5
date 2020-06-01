package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/axgle/mahonia"
)

var enc mahonia.Encoder

// GetExecutablePath Get current executable's real path
func GetExecutablePath() (path string, dir string) {
	path, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir = filepath.Dir(path)

	return
}

// GetCurrentPath Get current program running directory
func GetCurrentPath() string {
	// dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	dir, err := filepath.Abs("./")
	if err != nil {
		log.Fatal(err)
	}

	return dir
}

// MD5Bytes Calculation MD5 value of a list of bytes
func MD5Bytes(s []byte) string {
	ret := md5.Sum(s)
	return hex.EncodeToString(ret[:])
}

// MD5File Calculation MD5 value of a file
func MD5File(file string) (string, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return MD5Bytes(data), nil
}

func calc(current string, path string, md5Chan chan string, coreChan chan int) {
	md5, _ := MD5File(path)
	value := md5 + " " + strings.Replace(path, current+"/", "", 1) + "\r\n"
	<-coreChan
	md5Chan <- enc.ConvertString(value)
	fmt.Print(value)
}

func main() {
	var current = filepath.ToSlash(GetCurrentPath())
	var count = 0
	var name = path.Base(current)
	var coreNum = runtime.NumCPU()

	var md5Chan = make(chan string)
	var routineChan = make(chan int, coreNum)

	enc = mahonia.NewEncoder("gbk")

	err := filepath.Walk(current, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		count++
		routineChan <- -1

		go calc(current, filepath.ToSlash(path), md5Chan, routineChan)

		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	fp, err := os.OpenFile(path.Join(current, name+".md5"), os.O_CREATE|os.O_WRONLY, 0777)
	defer fp.Close()

	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < count; i++ {
		value := <-md5Chan
		fp.WriteString(value)
	}
}
