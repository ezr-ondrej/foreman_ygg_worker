package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"git.sr.ht/~spc/go-log"
	"github.com/google/uuid"
	"github.com/redhatinsights/yggdrasil/worker"
)

// workerRx implements the startScript functionality in a function handler
// compatible with the yggdrasil/worker package API.
func workerRx(w *worker.Worker, _ string, id string, _ string, metadata map[string]string, data []byte) error {
	started := func(pid int, stdout, stderr io.ReadCloser) {
		pipe := func(outfd string, r io.ReadCloser) {
			buf := make([]byte, 4096)
			n, err := r.Read(buf)
			if err != nil {
				log.Errorf("cannot read buffer: %v", err)
			}
			if n > 0 {
				output := strings.TrimRight(string(buf), "\n\x00")
				content := struct {
					Output string `json:"output"`
					Type   string `json:"type"`
				}{
					Output: output,
					Type:   outfd,
				}
				log.Tracef("%v message: %v", outfd, output)
				body, err := json.Marshal(content)
				if err != nil {
					log.Errorf("cannot marshal content: %v", err)
				}
				_, _, _, err = w.Transmit(metadata["return_url"], uuid.New().String(), id, nil, body)
				if err != nil {
					log.Errorf("transmit failed: %v", err)
				}
			}
		}

		go pipe("stdout", stdout)
		go pipe("stderr", stderr)

		stopped := func(pid int, state *os.ProcessState) {
			_, _, _, err := w.Transmit(metadata["return_url"], uuid.New().String(), id, nil, []byte(fmt.Sprintf("{\"exit_code\":%v}", state.ExitCode())))
			if err != nil {
				log.Errorf("transmit failed: %v", err)
			}
		}

		if err := WaitProcess(pid, stopped); err != nil {
			log.Errorf("failed to wait for process: %v", err)
		}
	}

	if err := runScript(data, started); err != nil {
		return fmt.Errorf("cannot run script: %v", err)
	}

	return nil
}

// runScript writes the given script to a temporary file and executes it in a
// subshell. The started function is called after the process starts.
func runScript(script []byte, started ProcessStartedFunc) error {
	file, err := os.CreateTemp("/tmp", "ygg_rex")
	if err != nil {
		return fmt.Errorf("failed to create script tmp file: %v", err)
	}

	written, err := file.Write(script)
	if err != nil {
		return fmt.Errorf("failed to write script to tmp file: %v", err)
	}
	log.Debugf("script of %d bytes written in: %#v", written, file.Name())

	if err := file.Close(); err != nil {
		return fmt.Errorf("cannot close file: %v", err)
	}

	err = os.Chmod(file.Name(), 0700)
	if err != nil {
		return fmt.Errorf("cannot set permissions on file: %v", err)
	}

	return StartProcess("/bin/sh", []string{"-c", file.Name()}, []string{}, func(pid int, stdout, stderr io.ReadCloser) {
		started(pid, stdout, stderr)
		os.Remove(file.Name())
	})
}
