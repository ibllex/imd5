package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
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
var dec mahonia.Decoder

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

func isFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func calc(current string, path string, md5Chan chan string, coreChan chan int) {
	md5, _ := MD5File(path)
	value := md5 + " " + strings.Replace(path, current+"/", "", 1) + "\r\n"
	<-coreChan
	md5Chan <- enc.ConvertString(value)
	fmt.Print(value)
}

func check(sum string, path string, sumChan chan string, coreChan chan int) {
	result := ""

	if ok, _ := isFileExists(path); ok {
		md5, _ := MD5File(path)
		if strings.ToUpper(md5) == strings.ToUpper(sum) {
			result += "[ √ ]"
		} else {
			result += "[ ✗ ]"
		}
	} else {
		result += "[ ! ]"
	}

	result += " " + path
	<-coreChan
	sumChan <- enc.ConvertString(result)
	fmt.Println(result)
}

func checkMD5(sumFile string, routineChan chan int, sumChan chan string) {
	var count = 0

	fp, err := os.OpenFile(sumFile, os.O_RDONLY, 0777)
	if err != nil {
		fmt.Println("Could not open file " + sumFile)
		return
	}
	defer fp.Close()

	br := bufio.NewReader(fp)
	for {
		data, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}

		line := strings.Fields(dec.ConvertString(string(data)))
		if len(line) == 2 {
			count++
			routineChan <- -1
			go check(line[0], line[1], sumChan, routineChan)
		}
	}

	for i := 0; i < count; i++ {
		value := <-sumChan
		fp.WriteString(value)
	}
}

func sumMD5(routineChan chan int, md5Chan chan string) {
	var count = 0
	var current = filepath.ToSlash(GetCurrentPath())
	var name = path.Base(current)

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

func main() {
	var sumFile string
	var coreNum = runtime.NumCPU()

	var md5Chan = make(chan string)
	var routineChan = make(chan int, coreNum)

	flag.StringVar(&sumFile, "c", "", "read MD5 sums from the FILEs and check them")
	flag.Parse()

	enc = mahonia.NewEncoder("gbk")
	dec = mahonia.NewDecoder("gbk")

	if sumFile != "" {
		checkMD5(sumFile, routineChan, md5Chan)
	} else {
		sumMD5(routineChan, md5Chan)
	}
}
