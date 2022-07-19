package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)

func Post(body []byte, url string, cookie string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", cookie)
	req.Header.Add("Content-Type", "application/json")
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	rspBody, err := ioutil.ReadAll(rsp.Body)
	return rspBody, err
}

func Put(url string, cookie string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", cookie)
	req.Header.Add("accept", "*/*")
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	rspBody, err := ioutil.ReadAll(rsp.Body)
	return rspBody, err
}

func Get(url string, cookie string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", cookie)
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	rspBody, err := ioutil.ReadAll(rsp.Body)
	return rspBody, err
}

func Delete(url string, cookie string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", cookie)
	req.Header.Add("accept", "*/*")
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	rspBody, err := ioutil.ReadAll(rsp.Body)
	return rspBody, err
}

func GetFile(url string, filePath string, cookie string) error {

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Cookie", cookie)
	rsp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("下载文件时接口请求异常: %v", err)
	}

	defer rsp.Body.Close()

	// 创建一个文件用于保存
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("下载文件时创建文件异常: %v", err)
	}
	defer out.Close()

	// 然后将响应流和文件流对接起来
	if _, err = io.Copy(out, rsp.Body); err != nil {
		return fmt.Errorf("下载文件时对接异常: %v", err)
	}
	return err
}

func PostFile(url string, filePath string, body []byte, cookie string) ([]byte, error) {
	// create body
	contType, reader, err := createFileBuffer(filePath, body)
	if err != nil {
		fmt.Println("文件流转换失败:", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", url, reader)

	if err != nil {
		fmt.Println("Post 请求发送失败")
		return nil, err
	}

	// add headers
	req.Header.Add("Content-Type", contType)
	req.Header.Add("Cookie", cookie)
	req.Header.Add("accept", "*/*")

	client := &http.Client{}
	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println("request send error:", err)
		return nil, err
	}
	defer rsp.Body.Close()
	return ioutil.ReadAll(rsp.Body)
}

func createFileBuffer(filePath string, body []byte) (string, io.Reader, error) {
	var err error

	var jsonData map[string]interface{}

	if err := json.Unmarshal(body, &jsonData); err != nil {
		return "", nil, err
	}

	// fmt.Printf("%+v\n", jsonData)

	buf := new(bytes.Buffer)
	bw := multipart.NewWriter(buf) // body writer

	f, err := os.Open(filePath)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	for k, v := range jsonData {
		p1w, _ := bw.CreateFormField(k)
		p1w.Write(transformByte(v))
	}

	// file part1
	_, fileName := filepath.Split(filePath)
	fw1, _ := bw.CreateFormFile("file", fileName)
	io.Copy(fw1, f)

	bw.Close() //write the tail boundry
	return bw.FormDataContentType(), buf, nil
}

func transformByte(value interface{}) []byte {
	k := reflect.TypeOf(value).Kind()
	// fmt.Println(k)
	switch k {
	case reflect.String:
		// 将interface转为string字符串类型
		// fmt.Println("value type is string")
		return []byte(value.(string))
	case reflect.Bool:
		// 将interface转为string字符串类型
		// fmt.Println("value type is bool")
		return []byte(fmt.Sprintf("%v", value.(bool)))
	case reflect.Int32:
		// 将interface转为int32类型
		// fmt.Println("value type is int32")
		return []byte(fmt.Sprintf("%v", value.(int32)))
	case reflect.Int64:
		// 将interface转为int64类型
		// fmt.Println("value type is int64")
		return []byte(fmt.Sprintf("%v", value.(int64)))
	case reflect.Float32:
		// 将interface转为int64类型
		// fmt.Println("value type is float32")
		return []byte(fmt.Sprintf("%v", value.(float32)))
	case reflect.Float64:
		// 将interface转为int64类型
		// fmt.Println("value type is float64")
		return []byte(fmt.Sprintf("%v", value.(float64)))
		// case []int:
		// 	// 将interface转为切片类型
		// 	fmt.Println("value type is Test []int")
		// 	return []byte(value.([]int))
	default:
		fmt.Println("unknown")
	}
	return nil
}
