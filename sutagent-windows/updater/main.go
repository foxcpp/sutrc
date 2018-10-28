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
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/foxcpp/sutrc/agent"
)

const svcname = "sutupdate"
const dispName = "State University of Telecommunications Agent Updater"
const description = "Performs sutagent service rotation and updates agent executable by server request."

var baseURL string

func main() {
	time.Sleep(time.Second * 3)
	client := agent.NewClient(baseURL)
	const executable = "C:\\Windows\\sutagent.exe"
	exec.Command("cmd", "/C", "taskkill /im sutagent.exe").Run()
	out, err := os.Create(executable)
	if err != nil {
		log.Fatalln(1, "Failed to download sutagent update:", err.Error())
		return
	}

	inp, err := client.Download(baseURL + "/sutagent.exe")
	if err != nil {
		log.Fatalln(1, "Failed to download sutagent update:", err.Error())
		return
	}

	_, err = io.Copy(out, inp)
	if err != nil {
		log.Fatalln(1, "Failed to download sutagent update:", err.Error())
		return
	}
	out.Close()

	err = exec.Command(executable).Start()
	if err != nil {
		log.Fatalf("failed to start agent executable: %v", err)
	}
}
