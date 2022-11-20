package go_deploy

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

func PrintNow() {
	fmt.Println(time.Now())
}

func DeployForUbuntu(username string, password string, serverIp string, serverPort string, fileName string, remotePath string, runPort string, runProfile string) {
	//生成文件
	genLocalFile(fileName)

	//连接服务器
	client := connectServer(username, password, serverIp, serverPort)
	defer client.Close()

	//赋予远程文件夹可读写权限
	runCommand(client, "sudo chmod 777 "+remotePath)

	//如果存在则删除文件
	res1 := runCommand(client, "ls -l "+remotePath+"/"+fileName)
	if strings.Index(res1, "cannot access") == -1 {
		runCommand(client, "cd "+remotePath+"; sudo rm -f "+fileName)
	}

	//上传文件
	uploadFile(client, fileName, remotePath)

	//赋予新上传的文件可执行权限
	runCommand(client, "sudo chmod 777 "+remotePath+"/"+fileName)

	//停止目前的程序
	res2 := runCommand(client, "ps -ef | grep \""+fileName+"\" | grep -v \"grep\"")
	if res2 != "" {
		arr := strings.Fields(res2)
		runCommand(client, "sudo kill -9 "+arr[1])
	}

	//启动远程服务器上的项目
	runCommand(client, "cd "+remotePath+"; nohup ./"+fileName+" -port="+runPort+" -profile="+runProfile+" >/dev/null 2>&1 &")

	//删除本地文件
	delLocalFile(fileName)

	//最新信息
	res3 := runCommand(client, "ps -ef | grep \""+fileName+"\" | grep -v \"grep\"")
	if res3 != "" {
		arr := strings.Fields(res3)
		fmt.Println("最新pid " + arr[1])
	}
}

func genLocalFile(fileName string) {
	exec.Command("powershell", "$env:CGO_ENABLED=0;$env:GOOS=\"linux\";$env:GOARCH=\"amd64\";go build -ldflags=\"-s -w\" "+fileName+".go").Run()
	fmt.Println("已生成文件 " + fileName)
}

func delLocalFile(fileName string) {
	exec.Command("powershell", "rm "+fileName).Run()
	fmt.Println("已删除文件 " + fileName)
}

func runCommand(client *ssh.Client, cmd string) string {
	s1, _ := client.NewSession()
	defer s1.Close()
	buf, _ := s1.CombinedOutput(cmd)

	fmt.Println("已执行命令 " + cmd)
	return string(buf)
}

func connectServer(username string, password string, serverIp string, serverPort string) *ssh.Client {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, _ := ssh.Dial("tcp", serverIp+":"+serverPort, config)
	fmt.Println("已连接服务器 " + serverIp)
	return client
}

func uploadFile(client *ssh.Client, fileNamePath string, remotePath string) {
	sftpClient, _ := sftp.NewClient(client)
	defer sftpClient.Close()

	//打开本地文件流
	srcFile, err := os.Open(fileNamePath)
	if err != nil {
		fmt.Println("os.Open error : ", fileNamePath)
		log.Fatal(err)

	}
	//关闭文件流
	defer srcFile.Close()
	//上传到远端服务器的文件名,与本地路径末尾相同
	var remoteFileName = path.Base(fileNamePath)

	fmt.Println("正上传文件 " + path.Join(remotePath, remoteFileName) + " 请稍后...")

	//打开远程文件,如果不存在就创建一个
	dstFile, err := sftpClient.Create(path.Join(remotePath, remoteFileName))
	if err != nil {
		fmt.Println("sftpClient.Create error : ", path.Join(remotePath, remoteFileName))
		log.Fatal(err)

	}
	//关闭远程文件
	defer dstFile.Close()
	//读取本地文件,写入到远程文件中(这里没有分快穿,自己写的话可以改一下,防止内存溢出)
	ff, err := ioutil.ReadAll(srcFile)
	if err != nil {
		fmt.Println("ReadAll error : ", fileNamePath)
		log.Fatal(err)

	}
	dstFile.Write(ff)

	fmt.Println("已上传文件 " + path.Join(remotePath, remoteFileName))
}
