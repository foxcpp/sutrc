/* MIT License
 *
 * Copyright (c) 2018  Vladyslav Yamkovyi (Hexawolf)
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to
 * deal in the Software without restriction, including without limitation the
 * rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */
package main

import (
	"github.com/foxcpp/sutrc/agent"
	"github.com/kbinani/screenshot"
	"image/jpeg"
	"image/png"
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

func screenshotTask(client *agent.Client, taskID int, _ map[string]interface{}) {
	img, err := screenshot.CaptureDisplay(0)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{
			"error": true,
			"msg": err.Error(),
		})
		return
	}

	// Use pipe to avoid copying; blocks task thread until wtr.Close() is called (encoding finished)
	rdr, wtr := io.Pipe()
	go func() {
		jpeg.Encode(wtr, img, &jpeg.Options{
			Quality: 50,
		})
		png.Encode(wtr, img)
		wtr.Close()
	}()
	url, err := client.UploadFile(rdr)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{
			"error": true,
			"msg": err.Error(),
		})
		return
	}

	client.SendTaskResult(taskID, map[string]interface{}{
		"url": url,
	})
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
