package processrunner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const OutputLimitBytes = 256 * 1024

type StartRequest struct {
	Command    string
	WorkingDir string
}

type Record struct {
	ID                  string
	Command             string
	WorkingDir          string
	StartedAtMS         int64
	Running             bool
	ExitCode            *int
	RetainedOutputBytes int
	OutputTruncated     bool
}

type Service struct {
	mu        sync.Mutex
	next      int
	processes map[string]*process
}

type process struct {
	record Record
	cmd    *exec.Cmd
	output outputBuffer
}

func NewService() *Service {
	return &Service{processes: map[string]*process{}}
}

func (s *Service) Start(req StartRequest) (Record, error) {
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return Record{}, fmt.Errorf("command required")
	}
	workingDir, err := resolveWorkingDir(req.WorkingDir)
	if err != nil {
		return Record{}, err
	}
	cmd := shellCommand(command)
	cmd.Dir = workingDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Record{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Record{}, err
	}
	s.mu.Lock()
	s.next++
	record := Record{
		ID:          fmt.Sprintf("process_%06d", s.next),
		Command:     command,
		WorkingDir:  workingDir,
		StartedAtMS: time.Now().UnixMilli(),
		Running:     true,
	}
	proc := &process{record: record, cmd: cmd, output: newOutputBuffer(OutputLimitBytes)}
	proc.output.append([]byte(fmt.Sprintf("command: %s\ncwd: %s\n\n", command, workingDir)))
	s.processes[record.ID] = proc
	s.mu.Unlock()
	if err := cmd.Start(); err != nil {
		return Record{}, err
	}
	go s.capture(record.ID, stdout)
	go s.capture(record.ID, stderr)
	go s.wait(record.ID, cmd)
	return record, nil
}

func (s *Service) Output(id string, maxBytes int) (string, Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	proc, ok := s.processes[id]
	if !ok {
		return "", Record{}, fmt.Errorf("process not found")
	}
	return proc.output.snapshot(maxBytes), proc.recordWithOutput(), nil
}

func (s *Service) List() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, 0, len(s.processes))
	for _, proc := range s.processes {
		out = append(out, proc.recordWithOutput())
	}
	return out
}

func (s *Service) Kill(id string) (Record, error) {
	s.mu.Lock()
	proc, ok := s.processes[id]
	if !ok {
		s.mu.Unlock()
		return Record{}, fmt.Errorf("process not found")
	}
	cmd := proc.cmd
	s.mu.Unlock()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	proc.record.Running = false
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		proc.record.ExitCode = &code
	}
	return proc.recordWithOutput(), nil
}

func (s *Service) Shutdown() {
	for _, record := range s.List() {
		if record.Running {
			_, _ = s.Kill(record.ID)
		}
	}
}

func (s *Service) capture(id string, reader io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			s.mu.Lock()
			if proc, ok := s.processes[id]; ok {
				proc.output.append(buf[:n])
			}
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (s *Service) wait(id string, cmd *exec.Cmd) {
	err := cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	proc, ok := s.processes[id]
	if !ok || !proc.record.Running {
		return
	}
	proc.record.Running = false
	code := 0
	if err != nil && cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	proc.record.ExitCode = &code
}

func (p *process) recordWithOutput() Record {
	record := p.record
	record.RetainedOutputBytes = p.output.len()
	record.OutputTruncated = p.output.truncated
	return record
}

type outputBuffer struct {
	bytes     []byte
	limit     int
	truncated bool
}

func newOutputBuffer(limit int) outputBuffer {
	return outputBuffer{limit: limit}
}

func (b *outputBuffer) append(chunk []byte) {
	b.bytes = append(b.bytes, chunk...)
	if b.limit > 0 && len(b.bytes) > b.limit {
		b.bytes = append([]byte(nil), b.bytes[len(b.bytes)-b.limit:]...)
		b.truncated = true
	}
}

func (b *outputBuffer) snapshot(maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if maxBytes >= len(b.bytes) {
		return string(b.bytes)
	}
	return string(b.bytes[len(b.bytes)-maxBytes:])
}

func (b *outputBuffer) len() int {
	return len(b.bytes)
}

func resolveWorkingDir(workingDir string) (string, error) {
	if strings.TrimSpace(workingDir) != "" {
		return workingDir, nil
	}
	return os.Getwd()
}

func shellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", command)
	}
	return exec.Command("sh", "-lc", command)
}
