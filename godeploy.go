package godeploy

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
)

func getAllFiles(fileName string, dirList []string) []string {
	var allFiles []string
	allFiles = append(allFiles, fileName)
	for i := 0; i < len(dirList); i++ {
		files, _ := GetAllFiles(dirList[i])
		allFiles = append(allFiles, files...)
	}

	return allFiles
}

//runParam just like "-port=3000 -profile=dev >/dev/null 2>&1"
//full cmd is: nohup fileName -port=3000 -profile=dev >/dev/null 2>&1 &
func DeployForUbuntu(username string, password string, serverIp string, serverPort string, fileName string, remotePath string, runParam string, dirList []string) {
	if remotePath == "" {
		panic("远程路径不能为空")
	}

	//本地所有文件，包含主文件，以及配置文件，模板文件，静态文件等等
	allFiles := getAllFiles(fileName, dirList)

	//生成主文件
	genLocalFile(fileName)

	//连接服务器
	client := connectServer(username, password, serverIp, serverPort)
	defer client.Close()

	//删除目录，建目录，设置777权限
	runCommand(client, strings.Join([]string{
		"sudo rm -rf " + remotePath,
		"sudo mkdir " + remotePath,
		"sudo chmod 777 " + remotePath,
	}, " ; "))

	for i := 0; i < len(allFiles); i++ {
		uploadFileNew(client, allFiles[i], remotePath)
	}

	//赋予新上传的文件可执行权限
	runCommand(client, "sudo chmod 777 "+remotePath+"/"+fileName)

	//停止目前的程序
	res2 := runCommand(client, "ps -ef | grep \""+fileName+"\" | grep -v \"grep\"")
	if res2 != "" {
		arr := strings.Fields(res2)
		runCommand(client, "sudo kill -9 "+arr[1])
	}

	//启动远程服务器上的项目
	runCommand(client, "cd "+remotePath+"; nohup ./"+fileName+" "+runParam+" &")

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
		fmt.Println("-------上传错误----")
		fmt.Println(err)
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
func uploadFileNew(client *ssh.Client, fileNamePath string, remotePath string) {
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
	//var remoteFileName = path.Base(fileNamePath)
	var remoteFileName = fileNamePath
	pathArr := strings.Split(remoteFileName, "/")
	if len(pathArr) > 1 {
		for i := 0; i < len(pathArr)-1; i++ {
			runCommand(client, strings.Join([]string{
				"sudo mkdir " + path.Join(remotePath, pathArr[i]),
				"sudo chmod 777 " + path.Join(remotePath, pathArr[i]),
			}, " ; "))
		}
	}

	//fmt.Println("正上传文件 " + path.Join(remotePath, remoteFileName) + " 请稍后...")

	//打开远程文件,如果不存在就创建一个
	dstFile, err := sftpClient.Create(path.Join(remotePath, remoteFileName))
	if err != nil {
		fmt.Println("sftpClient.Create error : ", path.Join(remotePath, remoteFileName))
		fmt.Println("-------上传错误----")
		fmt.Println(err)
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

//获取指定目录下的所有文件,包含子目录下的文件
func GetAllFiles(dirPth string) (files []string, err error) {
	var dirs []string
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}

	PthSep := string(os.PathSeparator)
	for _, fi := range dir {
		if fi.IsDir() { // 目录, 递归遍历
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			GetAllFiles(dirPth + PthSep + fi.Name())
		} else {
			files = append(files, dirPth+"/"+fi.Name())
		}
	}

	// 读取子目录下文件
	for _, table := range dirs {
		temp, _ := GetAllFiles(table)
		for _, temp1 := range temp {
			files = append(files, temp1)
		}
	}

	return files, nil
}
