package gitwrap

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/gookit/color"
	"github.com/gookit/goutil/fsutil"
)

// from: https://github.com/github/hub/blob/master/cmd/cmd.go

var (
	DefaultBin = "git"
	GitDir = ".git"
)

// GitWrap is a project-wide struct that represents a command to be run in the console.
type GitWrap struct {
	// Bin git bin name. default is "git"
	Bin string
	// Cmd sub command name of git
	// Cmd  string
	Args []string
	// extra
	WorkDir string
	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
	// inner
	gitDir string
}

// New instance
func New(args ...string) *GitWrap {
	return &GitWrap{
		Bin:    DefaultBin,
		// Cmd:    cmd,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (gw *GitWrap) String() string {
	args := make([]string, len(gw.Args))
	for i, a := range gw.Args {
		if strings.ContainsRune(a, '"') {
			args[i] = fmt.Sprintf(`'%s'`, a)
		} else if a == "" || strings.ContainsRune(a, '\'') || strings.ContainsRune(a, ' ') {
			args[i] = fmt.Sprintf(`"%s"`, a)
		} else {
			args[i] = a
		}
	}
	return fmt.Sprintf("%s %s", gw.Bin, strings.Join(args, " "))
}

// WithWorkDir returns the current object
func (gw *GitWrap) WithWorkDir(dir string) *GitWrap {
	gw.WorkDir = dir
	return gw
}

// WithStdin returns the current argument
func (gw *GitWrap) WithStdin(in *os.File) *GitWrap {
	gw.Stdin = in
	return gw
}

// WithOutput returns the current argument
func (gw *GitWrap) WithOutput(out *os.File, errOut *os.File) *GitWrap {
	gw.Stdout = out
	if errOut != nil {
		gw.Stderr = errOut
	}
	return gw
}

// SubCmd returns the current object
func (gw *GitWrap) SubCmd(cmd string) *GitWrap {
	gw.Args = append(gw.Args, cmd)
	return gw
}

// WithArg returns the current argument
func (gw *GitWrap) WithArg(args ...string) *GitWrap {
	gw.Args = append(gw.Args, args...)
	return gw
}

// WithArgs for the git
func (gw *GitWrap) WithArgs(args []string) *GitWrap {
	gw.Args = append(gw.Args, args...)
	return gw
}

// Output run and return output
func (gw *GitWrap) Output() (string, error) {
	verboseLog(gw)
	c := exec.Command(gw.Bin, gw.Args...)
	c.Stderr = gw.Stderr
	output, err := c.Output()

	return string(output), err
}

// CombinedOutput run and return output, will combine stderr and stdout output
func (gw *GitWrap) CombinedOutput() (string, error) {
	verboseLog(gw)
	output, err := exec.Command(gw.Bin, gw.Args...).CombinedOutput()

	return string(output), err
}

// IsGitRepo return the work dir is an git repo.
func (gw *GitWrap) IsGitRepo() bool {
	return fsutil.IsDir(gw.WorkDir + "/" + GitDir)
}

// GitDir return git data dir
func (gw *GitWrap) GitDir() string {
	if gw.gitDir != "" {
		return gw.gitDir
	}

	if gw.WorkDir != "" {
		gw.gitDir = gw.WorkDir + "/.git"
	} else {
		gw.gitDir = GitDir
	}
	return gw.gitDir
}

// CurrentBranch return current branch name
func (gw *GitWrap) CurrentBranch() string {
	// cat .git/HEAD
	// ref: refs/heads/fea_4_12
	return ""
}

// Success run and return whether success
func (gw *GitWrap) Success() bool {
	verboseLog(gw)
	err := exec.Command(gw.Bin, gw.Args...).Run()
	return err == nil
}

// NewExecCmd create exec.Cmd from current cmd
func (gw *GitWrap) NewExecCmd() *exec.Cmd {
	// gw.parseBinArgs()

	// create exec.Cmd
	return exec.Command(gw.Bin, gw.Args...)
}

// Run runs command with `Exec` on platforms except Windows
// which only supports `Spawn`
func (gw *GitWrap) Run() error {
	if isWindows() {
		return gw.Spawn()
	}
	return gw.Exec()
}

// Spawn runs command with spawn(3)
func (gw *GitWrap) Spawn() error {
	verboseLog(gw)
	c := exec.Command(gw.Bin, gw.Args...)
	c.Stdin = gw.Stdin
	c.Stdout = gw.Stdout
	c.Stderr = gw.Stderr

	return c.Run()
}

// Exec runs command with exec(3)
// Note that Windows doesn't support exec(3): http://golang.org/src/pkg/syscall/exec_windows.go#L339
func (gw *GitWrap) Exec() error {
	verboseLog(gw)

	binary, err := exec.LookPath(gw.Bin)
	if err != nil {
		return &exec.Error{
			Name: gw.Bin,
			Err:  fmt.Errorf("command not found"),
		}
	}

	args := []string{binary}
	args = append(args, gw.Args...)

	return syscall.Exec(binary, args, os.Environ())
}

func verboseLog(cmd *GitWrap) {
	if debug {
		color.Comment.Println("> ", cmd.String())
	}
}

func isWindows() bool {
	return runtime.GOOS == "windows" || detectWSL()
}

var detectedWSL bool
var detectedWSLContents string

// https://github.com/Microsoft/WSL/issues/423#issuecomment-221627364
func detectWSL() bool {
	if !detectedWSL {
		b := make([]byte, 1024)
		f, err := os.Open("/proc/version")
		if err == nil {
			_, _ = f.Read(b) // ignore error
			f.Close()
			detectedWSLContents = string(b)
		}
		detectedWSL = true
	}
	return strings.Contains(detectedWSLContents, "Microsoft")
}
