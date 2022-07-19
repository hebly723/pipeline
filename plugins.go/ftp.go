package plugins

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func UploadFile(client *ssh.Client, localPath string, remoteDir string, remoteFileName string) error {
	ftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("创建ftp客户端失败:%v", err)
	}

	defer ftpClient.Close()

	// fmt.Println(localPath, remoteFileName)
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开文件失败:%v", err)
	}
	defer srcFile.Close()

	dstFile, e := ftpClient.Create(path.Join(remoteDir, remoteFileName))
	if e != nil {
		return fmt.Errorf("创建文件失败:%v", e)

	}
	defer dstFile.Close()

	buffer := make([]byte, 1024000)
	for {
		n, err := srcFile.Read(buffer)
		dstFile.Write(buffer[:n])
		//注意，由于文件大小不定，不可直接使用buffer，否则会在文件末尾重复写入，以填充1024的整数倍
		if err != nil {
			if err == io.EOF {
				fmt.Println("已读取到文件末尾")
				break
			} else {
				return fmt.Errorf("读取文件出错:%v", err)
			}
		}
	}
	return nil
}

func DownloadFile(client *ssh.Client, remotePath string, localDir string, localFilename string) error {
	ftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("创建ftp客户端失败:%v", err)
	}

	defer ftpClient.Close()

	srcFile, err := ftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("文件读取失败:%v", err)
	}
	defer srcFile.Close()

	dstFile, e := os.Create(path.Join(localDir, localFilename))
	if e != nil {
		return fmt.Errorf("文件创建失败:%v", e)
	}
	defer dstFile.Close()
	if _, err1 := srcFile.WriteTo(dstFile); err1 != nil {
		return fmt.Errorf("文件写入失败:%v", err1)
	}
	// fmt.Println("文件下载成功")
	return nil
}
