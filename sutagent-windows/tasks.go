package main

import (
	"github.com/foxcpp/sutrc/agent"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func deleteFileTask(client *agent.Client, taskID int, body map[string]interface{}) {
	path, ok := body["path"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "path should be string"})
		return
	}
	if err := os.RemoveAll(path); err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}
	client.SendTaskResult(taskID, map[string]interface{}{"error": false})
}

func moveFileTask(client *agent.Client, taskID int, body map[string]interface{}) {
	fromPath, ok := body["frompath"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "frompath should be string"})
		return
	}
	toPath, ok := body["topath"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "topath should be string"})
		return
	}

	if err := os.Rename(fromPath, toPath); err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}

	client.SendTaskResult(taskID, map[string]interface{}{"error": false})
}

func uploadFileTask(client *agent.Client, taskID int, body map[string]interface{}) {
	path, ok := body["path"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "path should be string"})
		return
	}

	file, err := os.Open(path)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}

	url, err := client.UploadFile(file)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Upload fail: " + err.Error()})
		return
	}

	client.SendTaskResult(taskID, map[string]interface{}{"error": false, "url": url})
}

func dirContentsTask(client *agent.Client, taskID int, body map[string]interface{}) {
	path, ok := body["dir"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "path should be string"})
		return
	}

	dirInfo, err := ioutil.ReadDir(path)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}

	res := []map[string]interface{}{}
	for _, entry := range dirInfo {
		res = append(res, map[string]interface{}{
			"name":     entry.Name(),
			"dir":      entry.IsDir(),
			"fullpath": filepath.Join(path, entry.Name()),
		})
	}
	client.SendTaskResult(taskID, map[string]interface{}{"error": false, "contents": res})
}

func downloadFileTask(client *agent.Client, taskID int, body map[string]interface{}) {
	url, ok := body["url"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "url should be string"})
		return
	}
	outPath, ok := body["out"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "out should be string"})
		return
	}

	remoteFile, err := client.Download(url)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Download fail: " + err.Error()})
		return
	}

	file, err := os.Create(outPath)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}

	_, err = io.Copy(file, remoteFile)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}
	client.SendTaskResult(taskID, map[string]interface{}{"error": false})
}

func proclistTask(client *agent.Client, taskID int, _ map[string]interface{}) {
	procs, err := Processes()
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{
			"error": true,
			"msg":   err.Error(),
		})
		return
	}
	var windowsArray []Window
	for _, v := range procs {
		windowsArray = append(windowsArray, Window{
			PID:  v.Pid(),
			Name: v.Executable(),
		})
	}
	responseMap := map[string]interface{}{
		"procs": windowsArray,
	}
	client.SendTaskResult(taskID, responseMap)
}

func executeCmdTask(client *agent.Client, taskID int, body map[string]interface{}) {
	dec := cmdEncoding.NewDecoder()

	log.Println("Received execute_cmd task", body)
	command, ok := body["cmd"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "cmd should be string"})
		return
	}
	if command == "" {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Empty command is not allowed"})
		return
	}

	out := exec.Command("cmd", "/C", command)
	returnResult, err := out.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
			return
		}
	}

	decodedOut, err := dec.String(string(returnResult))
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Can't convert output to Unicode"})
		return
	}

	client.SendTaskResult(taskID, map[string]interface{}{
		"status_code": out.ProcessState.Sys().(syscall.WaitStatus).ExitCode,
		"output":      decodedOut,
	})
}
