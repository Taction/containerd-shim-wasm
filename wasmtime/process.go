package wasmtime

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/console"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/pkg/stdio"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type process struct {
	mu sync.Mutex

	id         string
	exitStatus int
	exitTime   time.Time
	stdio      stdio.Stdio
	stdin      io.Closer
	process    *os.Process
	exited     chan struct{}
	ec         chan<- Exit

	env  []string
	args []string

	isSandbox bool

	waitError error
}

func (p *process) ID() string {
	return p.id
}

func (p *process) Pid() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.process != nil {
		return p.process.Pid
	}
	return 0
}

func (p *process) ExitStatus() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitStatus
}

func (p *process) ExitedAt() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitTime
}

func (p *process) Stdin() io.Closer {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stdin
}

func (p *process) Stdio() stdio.Stdio {
	return p.stdio
}

func (p *process) Status(context.Context) (string, error) {
	select {
	case <-p.exited:
	default:
		p.mu.Lock()
		running := p.process != nil
		p.mu.Unlock()
		if running {
			return "running", nil
		}
		return "created", nil
	}

	return "stopped", nil
}

func (p *process) Wait() {
	<-p.exited
}

func (p *process) Resize(ws console.WinSize) error {
	return nil
}

func (p *process) Start(ctx context.Context) (err error) {
	var args []string
	/*
		TODO: deal with envs and args
		for _, rm := range p.remaps {
			args = append(args, "--mapdir="+rm)
		}
		for _, env := range p.env {
			args = append(args, "--env="+env)
		}
	*/
	args = append(args, p.args...)
	cmd := exec.Command("wasmtime", p.args...)
	// If this is a sandbox, run a normal process
	if p.isSandbox {
		cmd = exec.Command(args[0])
		if len(args) > 1 {
			cmd = exec.Command(args[0], args[1:]...)
		}
	}

	var in io.Closer
	var closers []io.Closer
	if p.stdio.Stdin != "" {
		stdin, err := os.OpenFile(p.stdio.Stdin, os.O_RDONLY, 0)
		if err != nil {
			return errors.Wrapf(err, "unable to open stdin: %s", p.stdio.Stdin)
		}
		defer func() {
			if err != nil {
				stdin.Close()
			}
		}()
		cmd.Stdin = stdin
		in = stdin
		closers = append(closers, stdin)
	}

	if p.stdio.Stdout != "" {
		stdout, err := os.OpenFile(p.stdio.Stdout, os.O_WRONLY, 0)
		if err != nil {
			return errors.Wrapf(err, "unable to open stdout: %s", p.stdio.Stdout)
		}
		defer func() {
			if err != nil {
				stdout.Close()
			}
		}()
		cmd.Stdout = stdout
		closers = append(closers, stdout)
	}

	if p.stdio.Stderr != "" {
		stderr, err := os.OpenFile(p.stdio.Stderr, os.O_WRONLY, 0)
		if err != nil {
			return errors.Wrapf(err, "unable to open stderr: %s", p.stdio.Stderr)
		}
		defer func() {
			if err != nil {
				stderr.Close()
			}
		}()
		cmd.Stderr = stderr
		closers = append(closers, stderr)
	}

	p.mu.Lock()
	if p.process != nil {
		return errors.Wrap(errdefs.ErrFailedPrecondition, "already running")
	}
	if err := cmd.Start(); err != nil {
		p.mu.Unlock()
		return err
	}
	p.process = cmd.Process
	p.stdin = in
	p.mu.Unlock()

	log := log.GetLogger(context.TODO())
	log.Infof("wasm Start: %d", p.Pid())

	go func() {
		waitStatus, err := p.process.Wait()
		p.mu.Lock()
		p.exitTime = time.Now()
		if err != nil {
			p.exitStatus = -1
			logrus.WithError(err).Errorf("wait returned error")
		} else if waitStatus != nil {
			// TODO: Make this cross platform
			p.exitStatus = int(waitStatus.Sys().(syscall.WaitStatus))
		}
		p.mu.Unlock()

		close(p.exited)

		p.ec <- Exit{
			Pid:    p.Pid(),
			Status: p.exitStatus,
		}

		for _, c := range closers {
			c.Close()
		}
	}()

	return nil
}

func (p *process) Delete(context.Context) error {
	return nil
}

func (p *process) Kill(context.Context, uint32, bool) error {
	p.mu.Lock()
	running := p.process != nil
	p.mu.Unlock()

	if !running {
		return errors.New("not started")
	}

	err := p.process.Kill()
	if err != nil && err.Error() != "os: process already finished" {
		return err
	}
	return nil
}

func (p *process) SetExited(status int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.exitStatus = status
}
