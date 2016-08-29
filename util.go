package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/astaxie/beego/httplib"
)

type Common struct {
}

func (this *Common) GetArgsMap() map[string]string {

	return this.ParseArgs(strings.Join(os.Args, "$$$$"), "$$$$")

}

func (this *Common) Home() (string, error) {
	user, err := user.Current()
	if nil == err {
		return user.HomeDir, nil
	}

	if "windows" == runtime.GOOS {
		return this.homeWindows()
	}

	return this.homeUnix()
}

func (this *Common) homeUnix() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	// If that fails, try the shell
	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "eval echo ~$USER")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", errors.New("blank output when reading home directory")
	}

	return result, nil
}

func (this *Common) homeWindows() (string, error) {
	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	home := drive + path
	if drive == "" || path == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", errors.New("HOMEDRIVE, HOMEPATH, and USERPROFILE are blank")
	}

	return home, nil
}

func (this *Common) GetAllIps() []string {
	ips := []string{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, addr := range addrs {
		ip := addr.String()
		pos := strings.Index(ip, "/")
		if match, _ := regexp.MatchString("(\\d+\\.){3}\\d+", ip); match {
			if pos != -1 {
				ips = append(ips, ip[0:pos])
			}
		}
	}
	return ips
}

func (this *Common) GetLocalIP() string {

	ips := this.GetAllIps()
	for _, v := range ips {
		if strings.HasPrefix(v, "10.") || strings.HasPrefix(v, "172.") || strings.HasPrefix(v, "172.") {
			return v
		}
	}
	return "127.0.0.1"

}

func (this *Common) JsonEncode(v interface{}) string {

	if v == nil {
		return ""
	}
	jbyte, err := json.Marshal(v)
	if err == nil {
		return string(jbyte)
	} else {
		return ""
	}

}

func (this *Common) JsonDecode(jsonstr string) interface{} {

	var v interface{}
	err := json.Unmarshal([]byte(jsonstr), &v)
	if err != nil {
		return nil

	} else {
		return v
	}

}

func (this *Common) ParseArgs(args string, sep string) map[string]string {

	ret := make(map[string]string)

	var argv []string

	argv = strings.Split(args, sep)

	for i, v := range argv {
		if strings.HasPrefix(v, "-") && len(v) == 2 {
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
				ret[v[1:]] = argv[i+1]
			}
		}

	}
	for i, v := range argv {
		if strings.HasPrefix(v, "-") && len(v) == 2 {
			if i+1 < len(argv) && strings.HasPrefix(argv[i+1], "-") {
				ret[v[1:]] = "1"
			} else if i+1 == len(argv) {
				ret[v[1:]] = "1"
			}
		}

	}

	for i, v := range argv {
		if strings.HasPrefix(v, "--") && len(v) > 3 {
			if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "--") {
				ret[v[2:]] = argv[i+1]
			}
		}

	}
	for i, v := range argv {
		if strings.HasPrefix(v, "--") && len(v) > 3 {
			if i+1 < len(argv) && strings.HasPrefix(argv[i+1], "--") {
				ret[v[2:]] = "1"
			} else if i+1 == len(argv) {
				ret[v[2:]] = "1"
			}
		}

	}

	return ret

}

func (this *Common) MD5(str string) string {

	md := md5.New()
	md.Write([]byte(str))
	return fmt.Sprintf("%x", md.Sum(nil))
}

func (this *Common) GetHostName() string {

	return ""
}

func (this *Common) IsExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func (this *Common) ReadFile(path string) string {
	if this.IsExist(path) {
		fi, err := os.Open(path)
		if err != nil {
			return ""
		}
		defer fi.Close()
		fd, err := ioutil.ReadAll(fi)
		return string(fd)
	} else {
		return ""
	}
}

func (this *Common) WriteFile(path string, content string) bool {
	var f *os.File
	var err error
	if this.IsExist(path) {
		f, err = os.OpenFile(path, os.O_RDWR, 0666)

	} else {
		f, err = os.Create(path)

	}
	if err == nil {
		defer f.Close()
		if _, err = io.WriteString(f, content); err == nil {
			return true
		} else {
			return false
		}
	} else {
		return false
	}

}

func (this *Common) GetProductUUID() string {

	filename := "/sys/devices/virtual/dmi/id/product_uuid"
	uuid := this.ReadFile(filename)
	if uuid == "" {
		filename = "/etc/uuid"
		if this.IsExist(filename) {
			uuid = this.ReadFile(filename)
		} else {
			os.Mkdir(filename, 0666)
		}
		if uuid == "" {
			uuid := this.GetUUID()
			this.WriteFile(filename, uuid)

		}

	}

	return strings.Trim(uuid, "\n")

}

func (this *Common) Download(url string, data map[string]string) []byte {

	req := httplib.Post(url)
	for k, v := range data {
		req.Param(k, v)
	}
	str, err := req.Bytes()

	if err != nil {

		return nil

	} else {
		return str
	}
}

func (this *Common) Exec(cmd []string, timeout int) (string, int) {

	var out bytes.Buffer

	sig := syscall.SIGKILL

	duration := time.Duration(timeout) * time.Second

	command := exec.Command(cmd[0], cmd[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = &out
	command.Stderr = &out

	err := command.Start()
	if err != nil {
		//		die2(fmt.Sprintf("[timeout] Can't start the process: %v", err), 127)
	}

	timer := time.AfterFunc(duration, func() {
		//		if err := command.Process.Signal(sig); err != nil {
		//			fmt.Fprintf(os.Stderr, "[timeout] Can't kill the process: %v\n", err)
		//			command.Process.Release()
		//			print(command.Process.Pid)
		//		}
		print("killed processid", command.Process.Pid)
		command.Process.Kill()

	})

	err = command.Wait()

	killed := !timer.Stop()

	status := 0
	if killed {
		if sig == syscall.SIGKILL {
			status = 132
		} else {
			status = 124
		}
	} else if err != nil {
		if command.ProcessState == nil {
		}
		status = command.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	} else {
		status = 0
	}

	return out.String(), status

}

func (this *Common) GetUUID() string {

	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	id := this.MD5(base64.URLEncoding.EncodeToString(b))
	return fmt.Sprintf("%s-%s-%s-%s-%s", id[0:8], id[8:12], id[12:16], id[16:20], id[20:])

}

func (this *Common) Request(url string, data map[string]string) string {
	body := "{}"

	for _, k := range []string{"i", "s"} {
		if _, ok := data[k]; !ok {
			switch k {
			case "i":
				data[k] = this.GetLocalIP()
			case "s":
				data[k], _ = os.Hostname()
			}
		}

	}

	if pdata, err := json.Marshal(data); err == nil {
		body = string(pdata)
	}
	req := httplib.Post(url)
	if dir, ok := this.Home(); ok == nil {
		filename := dir + "/" + ".cli"
		uuid := this.ReadFile(filename)
		if uuid != "" {
			req.Header("auth-uuid", uuid)
		}
	}

	req.Param("param", body)
	req.SetTimeout(time.Second*10, time.Second*60)
	str, err := req.String()
	if err != nil {
		print(err)
	}
	return str
}
