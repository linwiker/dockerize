package main

import (
	"log"
	"opstools/dockerize/reaper"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
)

func runCmd(ctx context.Context, cancel context.CancelFunc, cmd string, args ...string) {
	defer wg.Done()
	reaper.Set()
	process := exec.Command(cmd, args...)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	// start the process
	err := process.Start()
	if err != nil {
		log.Fatalf("Error starting command: `%s` - %s\n", cmd, err)
	}

	// Setup signaling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGKILL, unix.SIGCHLD, unix.SIGPIPE)

	wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("received goroutine panic reason=%v", r)
			}
		}()
		defer wg.Done()
		for {
			select {
			case sig := <-sigs:
				switch sig {
				case unix.SIGCHLD:
					var status unix.WaitStatus
					var rus unix.Rusage
					flag := unix.WNOHANG
					for {
						pid, err := unix.Wait4(-1, &status, flag, &rus)
						if err != nil {
							if err == unix.ECHILD {
								break
							}
							log.Printf("Received SIGCHLD signal: %s error: %s\n", sig)
						}
						if pid <= 0 {
							break
						}
						log.Printf("Received SIGCHLD signal: %s; pid %v exit; status: %v\n", sig, pid, status)
					}
				case unix.SIGTERM, unix.SIGINT, unix.SIGHUP, unix.SIGKILL:
					log.Printf("Received SIGTERM: %s\n", sig)
					signalProcessWithTimeout(process, sig)
					cancel()
				case unix.SIGPIPE:
					log.Printf("Received SIGPIPE: %s\n", sig)
				}
			case <-ctx.Done():
				log.Printf("done.\n")
				return
				// exit when context is done
			}
		}
	}()

	err = process.Wait()
	cancel()

	if err == nil {
		log.Println("Command finished successfully.")
	} else {
		log.Printf("Command exited with error: %s\n", err)
		// OPTIMIZE: This could be cleaner
		os.Exit(err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus())
	}

}

func signalProcessWithTimeout(process *exec.Cmd, sig os.Signal) {
	done := make(chan struct{})

	go func() {
		process.Process.Signal(sig) // pretty sure this doesn't do anything. It seems like the signal is automatically sent to the command?
		process.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(10 * time.Second):
		log.Println("Killing command due to timeout.")
		process.Process.Kill()
	}
}
