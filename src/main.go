package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"git.sr.ht/~spc/go-log"

	"github.com/redhatinsights/yggdrasil/worker"
	pb "github.com/redhatinsights/yggdrasil_v0/protocol"
	"google.golang.org/grpc"
)

var yggdDispatchSocketAddr string

func main() {
	fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ExitOnError)
	lvl := fs.String("log-level", "error", "set log level to `LEVEL`")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("cannot parse flags: %v", err)
	}

	logLevel, err := log.ParseLevel(*lvl)
	if err != nil {
		log.Fatalf("cannot parse log-level: %v", err)
	}
	log.SetLevel(logLevel)

	// Get initialization values from the environment.
	var ok bool
	yggdDispatchSocketAddr, ok = os.LookupEnv("YGG_SOCKET_ADDR")
	if ok {
		log.Info("Detected YGG_SOCKET_ADDR environment variable; attempting gRPC connection")

		// Dial the dispatcher on its well-known address.
		conn, err := grpc.Dial(yggdDispatchSocketAddr, grpc.WithInsecure())
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		// Create a dispatcher client
		c := pb.NewDispatcherClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// Register as a handler of the "foreman" type.
		r, err := c.Register(ctx, &pb.RegistrationRequest{Handler: "foreman", Pid: int64(os.Getpid()), DetachedContent: true})
		if err != nil {
			log.Fatal(err)
		}
		if !r.GetRegistered() {
			log.Fatalf("handler registration failed: %v", err)
		}

		// Listen on the provided socket address.
		l, err := net.Listen("unix", r.GetAddress())
		if err != nil {
			log.Fatal(err)
		}

		// Register as a Worker service with gRPC and start accepting connections.
		s := grpc.NewServer()
		pb.RegisterWorkerServer(s, &foremanServer{})
		if err := s.Serve(l); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Info("No YGG_SOCKET_ADDR environment variable detected; attempting D-Bus connection")

		w, err := worker.NewWorker("foreman", true, nil, nil, workerRx, nil)
		if err != nil {
			log.Fatalf("cannot create worker: %v", err)
		}

		// Set up a channel to receive the TERM or INT signal over and clean up
		// before quitting.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

		if err := w.Connect(quit); err != nil {
			log.Fatalf("cannot connect: %v", err)
		}
	}
}
