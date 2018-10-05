/*
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
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"
)

const eventURL = "https://hexawolf.me/sutcontrol"

func longPoll(key string) {
	client := &http.Client{
		Timeout: 26,
	}

	for {
		resp, err := client.Get(eventURL + "?key=" + key)
		if err != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(30 * time.Second)
			continue
		}
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		bodyString := string(bodyBytes)
		resp.Body.Close()
		out := exec.Command("cmd", "/C", bodyString)
		err = out.Run()
		resultURL := eventURL + "?result="
		if err != nil {
			elog.Warning(1, "Failed to execute remote command: " + err.Error())
			resultURL += "FAILED"
		} else {
			resultURL += "SUCCESS"
		}
		client.Head(resultURL)
	}
}
