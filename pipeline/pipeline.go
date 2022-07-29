package pipeline

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hebly723/pipeline/functions"
	"github.com/hebly723/pipeline/plugins"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

var AssertArray []Assert

type Assert struct {
	Description string
	Assert      string
	Result      bool
}

type Config struct {
	Machines map[string]Machine `json:"machines,omitempty"`
	Tasks    []Task             `json:"tasks,omitempty"`
}

type Task struct {
	Machines []string `json:"machines,omitempty"`
	Atoms    []Atom   `json:"atoms,omitempty"`
}

type Atom struct {
	Description string        `json:"description,omitempty"`
	Loops       int           `json:"times,omitempty"`
	Timeout     int           `json:"timeout,omitempty"`
	Static      bool          `json:"static,omitempty"`
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
	Newf        string        `json:"newf,omitempty"`
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

func WaitEnter() {
	buf := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(sentence))
	}
}

func InitParams(params map[string]map[string]string, machines map[string]Machine) {
	for k := range machines {
		params[k] = make(map[string]string)
	}
}

func ReadConfig(dirname string) Config {
	st := Config{
		Machines: map[string]Machine{},
		Tasks:    []Task{},
	}

	files, err := ioutil.ReadDir(dirname)

	if err != nil {
		fmt.Printf("文件夹读取异常:%v\n", err)
	}

	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Name() < files[j].Name() {
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
	}
	return st
}

func (c *Config) Add(newConfig Config) {
	for k, v := range newConfig.Machines {
		c.Machines[k] = v
	}
	c.Tasks = append(c.Tasks, newConfig.Tasks...)
}

func DoPipeline(config Config, params map[string]map[string]string) error {
	for _, v := range config.Tasks {
		// fmt.Printf("%+v\n", v)
		for _, m := range v.Machines {
			c := config.Machines[m]
			if err := SSH_do(c.User, c.Password, c.Server, v.Atoms, params, m); err != nil {
				fmt.Println("出现错误:", err)
				return err
			}
		}
	}

	fmt.Println("用例\t预期\t实际执行结果")
	for _, v := range AssertArray {
		fmt.Printf("%v\t%v\t%v\n", v.Description, v.Assert, v.Result)
	}

	return nil
}

// 先输出，再使用map存储
func extractParams(result []byte, e ExtractParams, params map[string]string,
	description string) error {
	fmt.Printf("执行结果:%+v\n", string(result))
	if e.Type == "split" {
		middleResults := strings.Split(string(result), e.Separator)
		for _, v := range e.Result {
			fmt.Println("set", v.Name, middleResults[v.Index])
			params[v.Name] = middleResults[v.Index]
			if v.Assert != "" {
				if params[v.Name] != v.Assert {
					AssertArray = append(AssertArray, Assert{
						Description: description,
						Assert:      v.Assert,
						Result:      false,
					})
					return fmt.Errorf("与测试预期不符")
				} else {
					AssertArray = append(AssertArray, Assert{
						Description: description,
						Assert:      v.Assert,
						Result:      true,
					})
				}
			}
		}
		return nil
	}
	if e.Type == "json" {
		var middleResults map[string]interface{}
		if err := json.Unmarshal(result, &middleResults); err != nil {
			return fmt.Errorf("json转化时出错!错误为:%+v\n", err)
		}
		tempResults := middleResults
		for _, v := range e.Result {
			pure := v.Key
			k := strings.Split(pure, ".")
			for i := 0; i < len(k)-1; i++ {
				fmt.Print(k[i], tempResults[k[i]])
				tempResults = tempResults[k[i]].(map[string]interface{})
			}
			pure = k[len(k)-1]
			params[v.Name] = string(plugins.TransformInterfaceIntoByte(tempResults[pure]))
			if v.Assert != "" {
				if params[v.Name] != v.Assert {
					AssertArray = append(AssertArray, Assert{
						Description: description,
						Assert:      v.Assert,
						Result:      false,
					})
					return fmt.Errorf("与测试预期不符")
				} else {
					AssertArray = append(AssertArray, Assert{
						Description: description,
						Assert:      v.Assert,
						Result:      true,
					})
				}
			}

		}
	}
	return nil
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
		if err := v2.Work(params[machine], client); err != nil {
			fmt.Printf("执行原子时报错：%v\n", err)
		}
	}
	return nil
}

func (v2 *Atom) Work(params map[string]string, client *ssh.Client) error {
	fmt.Printf("atom: %+v\n", v2)
	beginTime := time.Now()
	result, err := v2.singleJob(params, client)
	if v2.Extract.Result != nil {
		extractParams(result, v2.Extract, params, v2.Description)
	}
	fmt.Println(string(result))
	if v2.Loops == 0 {
		return err
	} else if v2.Loops == -1 {
		for {
			<-time.After(time.Duration(v2.Timeout) * time.Millisecond)
			result, _ := v2.singleJob(params, client)
			if v2.Extract.Result != nil {
				if err := extractParams(result, v2.Extract, params, v2.Description); err == nil {
					break
				}
			}
		}
	} else {
		var wg sync.WaitGroup
		errCount := 0
		if err != nil {
			errCount++
		}
		wg.Add(v2.Loops - 1)
		for i := 1; i < v2.Loops; i++ {
			<-time.After(time.Duration(v2.Timeout) * time.Millisecond)
			go func() {
				defer wg.Done()
				result, err := v2.singleJob(params, client)
				if v2.Extract.Result != nil {
					if err := extractParams(result, v2.Extract, params, v2.Description); err != nil {
						errCount++
					}
				}
				if err != nil {
					errCount++
				}
			}()
		}
		wg.Wait()
		fmt.Printf("成功率: %v%%, 发生错误的次数: %v, 循环总数： %v\n", float64((v2.Loops-errCount)*100)/float64(v2.Loops), errCount, v2.Loops)

	}
	if v2.Static {
		fmt.Printf("总用时：%+v\n", (time.Now().Sub(beginTime)))
	}
	return nil
}

func (v2 *Atom) singleJob(params map[string]string, client *ssh.Client) ([]byte, error) {
	if v2.Plugin != "" && v2.Plugin != "cmd" {
		pluginID := strings.ToLower(v2.Plugin)
		if pluginID == "post" {
			return plugins.Post(
				[]byte(decodeString(v2.Body, params)),
				decodeString(v2.URL, params),
				decodeString(v2.Cookie, params),
			)
		}
		if pluginID == "get" {
			return plugins.Get(
				decodeString(v2.URL, params),
				decodeString(v2.Cookie, params))
		}
		if pluginID == "delete" {
			return plugins.Delete(
				decodeString(v2.URL, params),
				decodeString(v2.Cookie, params))
		}
		if pluginID == "put" {
			return plugins.Put(
				decodeString(v2.URL, params),
				decodeString(v2.Cookie, params))
		}
		if pluginID == "postfile" {
			return plugins.PostFile(
				decodeString(v2.URL, params),
				decodeString(v2.File, params),
				[]byte(decodeString(v2.Body, params)),
				decodeString(v2.Cookie, params))
		}
		if pluginID == "getfile" {
			return nil, plugins.GetFile(
				decodeString(v2.URL, params),
				decodeString(v2.File, params),
				decodeString(v2.Cookie, params))
		}
		if pluginID == "upload" {
			return nil, plugins.UploadFile(client,
				decodeString(v2.File, params),
				decodeString(v2.Dir, params),
				decodeString(v2.Newf, params))
		}
		if pluginID == "download" {
			return nil, plugins.DownloadFile(client,
				decodeString(v2.File, params),
				decodeString(v2.Dir, params),
				decodeString(v2.Newf, params))
		}
	}
	var result SshReturn
	if v2.Command != "" {
		result = ssh_session(*v2, client, params)
	}
	return result.data, nil
}

func ssh_session(atom Atom, client *ssh.Client, params map[string]string) SshReturn {
	var result SshReturn
	if session, err := client.NewSession(); err == nil {
		defer session.Close()
		session.Stdout = &result
		session.Stderr = os.Stderr
		commandStr := ""
		if atom.Path != "" {
			commandStr = commandStr + "cd " + atom.Path + " && "
		}
		session.Run(decodeString(commandStr+atom.Command, params))
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

func decodeString(str string, params map[string]string) string {
	result := decodeFunction(str)
	re := regexp.MustCompile(`\$\{(.*?)\}`)
	match := re.FindAllStringSubmatch(result, -1)
	for _, v := range match {
		result = strings.ReplaceAll(result, v[0], params[v[1]])
	}
	fmt.Println(result)
	return result
}

func decodeFunction(str string) string {
	re := regexp.MustCompile(`\$\{md5\{(.*?)\}\}`)
	match := re.FindAllStringSubmatch(str, -1)
	result := str
	for _, v := range match {
		result = strings.ReplaceAll(result, v[0], functions.GetFileMd5(v[1]))
	}

	re = regexp.MustCompile(`\$\{uuid\{(.*?)\}\}`)
	match = re.FindAllStringSubmatch(str, -1)
	result = str
	for _, v := range match {
		length, _ := strconv.Atoi(v[1])
		result = strings.ReplaceAll(result, v[0], functions.GetUUID(length))
	}
	return result
}
