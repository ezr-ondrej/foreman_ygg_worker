package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"time"

	"git.sr.ht/~spc/go-log"
	"github.com/google/uuid"
	pb "github.com/redhatinsights/yggdrasil/protocol"
	"google.golang.org/grpc"
)

func startScript(ctx context.Context, d *pb.Data) {
	script := string(d.GetContent())
	log.Tracef("running script : %#v", script)

	scriptfile, err := ioutil.TempFile("/tmp", "ygg_rex")
	if err != nil {
		log.Errorf("failed to create script tmp file: %v", err)
	}
	defer os.Remove(scriptfile.Name())

	n2, err := scriptfile.Write(d.GetContent())
	if err != nil {
		log.Errorf("failed to write script to tmp file: %v", err)
	}
	log.Debugf("script of %d bytes written in : %#v", n2, scriptfile.Name())

	err = scriptfile.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chmod(scriptfile.Name(), 0700)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", "-c", scriptfile.Name())
	// cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("cannot connect to stdout: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Errorf("cannot connect to stderr: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Errorf("cannot run script: %v", err)
		return
	}
	log.Infof("started script process: %v", cmd.Process.Pid)

	// Dial the Dispatcher
	conn, err := grpc.Dial(yggdDispatchSocketAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := pb.NewDispatcherClient(conn)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			msg := scanner.Text()
			log.Tracef("stdout message: %v", msg)
			sendUpdate(c, d.GetMessageId(), d.GetMetadata()["return_url"], msg, "stdout")
		}
		if err := scanner.Err(); err != nil {
			log.Errorf("cannot read from stdout: %v", err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			sendUpdate(c, d.GetMessageId(), d.GetMetadata()["return_url"], scanner.Text(), "stderr")
		}
		if err := scanner.Err(); err != nil {
			log.Errorf("cannot read from stderr: %v", err)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				sendExitCode(c, d.GetMessageId(), d.GetMetadata()["return_url"], status.ExitStatus())
			}
		} else {
			log.Errorf("script run failed: %v", err)
		}
	} else {
		sendExitCode(c, d.GetMessageId(), d.GetMetadata()["return_url"], 0)
	}
}

func sendUpdate(c pb.DispatcherClient, origmsgid string, url string, message string, stdtype string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	data := &pb.Data{
		MessageId:  uuid.New().String(),
		ResponseTo: origmsgid,
		Content:    []byte("{\"output\": \"" + message + "\", \"type\": \"" + stdtype + "\"}"),
		Directive:  url,
	}

	if _, err := c.Send(ctx, data); err != nil {
		log.Error(err)
	}
}

func sendExitCode(c pb.DispatcherClient, origmsgid string, url string, code int) {
	// wait for the other updates
	time.Sleep(time.Duration(2) * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	data := &pb.Data{
		MessageId:  uuid.New().String(),
		ResponseTo: origmsgid,
		Content:    []byte(fmt.Sprintf("{\"exit_code\": %d}", code)),
		Directive:  url,
	}

	if _, err := c.Send(ctx, data); err != nil {
		log.Error(err)
	}
}
