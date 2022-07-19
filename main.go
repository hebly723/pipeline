package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"pipeline/functions"
	"pipeline/plugins.go"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Machines map[string]Machine `json:"machines,omitempty"`
	Tasks    []Task             `json:"tasks,omitempty"`
}

type Task struct {
	Machines []string `json:"machines,omitempty"`
	Atoms    []Atom   `json:"atoms,omitempty"`
}

type Atom struct {
	Description string        `json:"description"`
	Path        string        `json:"path,omitempty"`
	Plugin      string        `json:"plugin,omitempty"`
	Body        string        `json:"body,omitempty"`
	File        string        `json:"file,omitempty"`
	URL         string        `json:"url,omitempty"`
	Cookie      string        `json:"cookie,omitempty"`
	Command     string        `json:"command,omitempty"`
	Wait        bool          `json:"wait,omitempty"`
	Extract     ExtractParams `json:"extract,omitempty"`
	Dir         string        `json:"dir,omitempty"`
	Newf        string        `json:"newf,omitempt"`
}

type ExtractParams struct {
	Type      string          `json:"type,omitempty"`
	Separator string          `json:"separator,omitempty"`
	Result    []ExtractResult `json:"result,omitempty"`
}

type ExtractResult struct {
	Index        int    `json:"index,omitempty"`
	Key          string `json:"key,omitempty"`
	Name         string `json:"name,omitempty"`
	Assert       string `json:"assert,omitempty"`
	AssertResult bool   `json:"assert_result,omitempty"`
}

type Machine struct {
	User     string `json:"user,omitempty"`
	Server   string `json:"server,omitempty"`
	Password string `json:"password,omitempty"`
}

func main() {
	params := make(map[string]map[string]string)
	config := readConfig("./")
	initParams(params, config.Machines)
	fmt.Printf("map初始化:%+v\n", params)
	doPipeline(config, params)
	waitEnter()
}

func waitEnter() {
	buf := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(sentence))
	}
}

func initParams(params map[string]map[string]string, machines map[string]Machine) {
	for k := range machines {
		params[k] = make(map[string]string)
	}
}

func readConfig(dirname string) Config {
	st := Config{
		Machines: map[string]Machine{},
		Tasks:    []Task{},
	}

	files, err := ioutil.ReadDir(dirname)

	if err != nil {
		fmt.Printf("文件夹读取异常:%v\n", err)
	}

	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Name() > files[j].Name() {
			return true
		}
		return false
	})

	for _, f := range files {
		var temp Config
		filename := f.Name()
		filenameSplitArray := strings.Split(filename, ".")
		if filenameSplitArray[len(filenameSplitArray)-1] != "yaml" && filenameSplitArray[len(filenameSplitArray)-1] != "yml" {
			continue
		}
		//读取yaml文件到缓存中
		config, err := ioutil.ReadFile(path.Join(dirname, f.Name()))
		if err != nil {
			fmt.Print(err)
		}
		//yaml文件内容影射到结构体中
		err1 := yaml.Unmarshal(config, &temp)
		if err1 != nil {
			fmt.Println("error", err1)
		}
		// 合并到结构体中
		st.Add(temp)
		fmt.Printf("%v\t读取成功!\n", f.Name())
	}

	// fmt.Printf("%+v\n", st)
	return st
}

func (c *Config) Add(newConfig Config) {
	for k, v := range newConfig.Machines {
		c.Machines[k] = v
	}
	c.Tasks = append(newConfig.Tasks, newConfig.Tasks...)
}

func doPipeline(config Config, params map[string]map[string]string) error {
	for _, v := range config.Tasks {
		fmt.Printf("%+v\n", v)
		for _, m := range v.Machines {
			c := config.Machines[m]
			if err := SSH_do(c.User, c.Password, c.Server, v.Atoms, params, m); err != nil {
				fmt.Println("出现错误:", err)
				return err
			}
		}
	}

	for _, v := range config.Tasks {
		fmt.Println("用例\t预期\t实际执行结果")
		for _, a := range v.Atoms {
			for _, r := range a.Extract.Result {
				if r.Assert != "" {
					fmt.Printf("%v\t%v\t%v\n", a.Description, r.Assert, r.AssertResult)
				}
			}
		}
	}

	return nil
}

// 先输出，再使用map存储
func extractParams(result []byte, e ExtractParams, params map[string]string) {
	if e.Type == "split" {
		middleResults := strings.Split(string(result), e.Separator)
		for _, v := range e.Result {
			fmt.Println("set", v.Name, middleResults[v.Index])
			params[v.Name] = middleResults[v.Index]
		}
		// fmt.Println(params)
		return
	}
	if e.Type == "json" {
		var middleResults map[string]string
		if err := json.Unmarshal(result, &middleResults); err != nil {
			fmt.Printf("json转化时出错!错误为:%+v\n", err)
		}
		for _, v := range e.Result {
			params[v.Name] = middleResults[v.Key]
			if v.Assert != "" {
				if middleResults[v.Key] != v.Assert {
					fmt.Println("与测试预期不符")
					v.AssertResult = false
				} else {
					v.AssertResult = true
				}
			}
		}
		// fmt.Println(params)
		return
	}
}

func SSH_do(user string, password string, ip_port string, atoms []Atom, params map[string]map[string]string, machine string) error {
	PassWd := []ssh.AuthMethod{ssh.Password(password)}
	Conf := ssh.ClientConfig{User: user, Auth: PassWd, HostKeyCallback: ssh.InsecureIgnoreHostKey()}
	client, err := ssh.Dial("tcp", ip_port, &Conf)
	if err != nil {
		fmt.Println("创建client失败", err)
		return err
	}
	defer client.Close()
	for _, v2 := range atoms {
		// fmt.Println(v2)
		var result SshReturn
		if v2.Plugin != "" && v2.Plugin != "cmd" {
			var rspBody []byte
			var err error
			if v2.Plugin == "POST" || v2.Plugin == "post" {
				rspBody, err = plugins.Post(
					[]byte(decodeString(v2.Body, params, machine)),
					decodeString(v2.URL, params, machine),
					decodeString(v2.Cookie, params, machine),
				)
			}
			if v2.Plugin == "GET" || v2.Plugin == "get" {
				rspBody, err = plugins.Get(
					decodeString(v2.URL, params, machine),
					decodeString(v2.Cookie, params, machine))
			}
			if v2.Plugin == "DELETE" || v2.Plugin == "delete" {
				rspBody, err = plugins.Delete(
					decodeString(v2.URL, params, machine),
					decodeString(v2.Cookie, params, machine))
			}
			if v2.Plugin == "PUT" || v2.Plugin == "put" {
				rspBody, err = plugins.Put(
					decodeString(v2.URL, params, machine),
					decodeString(v2.Cookie, params, machine))
			}
			if v2.Plugin == "POSTFILE" || v2.Plugin == "postfile" {
				rspBody, err = plugins.PostFile(
					decodeString(v2.URL, params, machine),
					decodeString(v2.File, params, machine),
					[]byte(decodeString(v2.Body, params, machine)),
					decodeString(v2.Cookie, params, machine))
			}
			if v2.Plugin == "GETFILE" || v2.Plugin == "getfile" {
				err = plugins.GetFile(
					decodeString(v2.URL, params, machine),
					decodeString(v2.File, params, machine),
					decodeString(v2.Cookie, params, machine))
			}
			if strings.ToLower(v2.Plugin) == "upload" {
				err = plugins.UploadFile(client,
					decodeString(v2.File, params, machine),
					decodeString(v2.Dir, params, machine),
					decodeString(v2.Newf, params, machine))
			}
			if strings.ToLower(v2.Plugin) == "download" {
				err = plugins.DownloadFile(client,
					decodeString(v2.File, params, machine),
					decodeString(v2.Dir, params, machine),
					decodeString(v2.Newf, params, machine))
			}
			if err != nil {
				return err
			}
			result.data = rspBody
		}

		result = ssh_session(v2, client, params, machine)
		fmt.Println(string(result.data))
		if v2.Extract.Result != nil {
			extractParams(result.data, v2.Extract, params[machine])
		}
	}
	return nil
}

func ssh_session(atom Atom, client *ssh.Client, params map[string]map[string]string, machine string) SshReturn {
	var result SshReturn
	if session, err := client.NewSession(); err == nil {
		defer session.Close()
		session.Stdout = &result
		session.Stderr = os.Stderr
		commandStr := ""
		if atom.Path != "" {
			commandStr = commandStr + "cd " + atom.Path + " && "
		}
		session.Run(decodeString(commandStr+atom.Command, params, machine))
		if atom.Wait {
			for result.data == nil {
				<-time.After(1 * time.Second)
			}
		}
	}
	return result
}

type SshReturn struct {
	data []byte
}

func (s *SshReturn) Write(p []byte) (n int, err error) {
	s.data = append(s.data, p...)
	return len(p), nil
}

// 正则表达式解码
func decodeString(str string, params map[string]map[string]string, machine string) string {
	re := regexp.MustCompile(`\$\{(.*?)\}`)
	match := re.FindAllStringSubmatch(str, -1)
	result := decodeFunction(str)
	// fmt.Printf("源:%+v, 匹配结果:%+v\n", str, match)
	for _, v := range match {
		result = strings.ReplaceAll(result, v[0], params[machine][v[1]])
	}
	fmt.Println(result)
	return result
}

func decodeFunction(str string) string {
	re := regexp.MustCompile(`\$\{md5\{(.*?)\}\}`)
	match := re.FindAllStringSubmatch(str, -1)
	result := str
	// fmt.Printf("源:%+v, 匹配结果:%+v\n", str, match)
	for _, v := range match {
		result = strings.ReplaceAll(result, v[0], functions.GetFileMd5(v[1]))
	}
	return result
}
