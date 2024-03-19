package gadgetconfig

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Action
type Action int

const (
	Noop Action = iota
	Comment
	Mkdir
	MkdirCreateOnly
	Rmdir
	Write
	WriteBinary
	Remove
	Symlink
)

// Step
type Step struct {
	Action Action
	Arg0   string
	Arg1   string
}

func (s Step) Run() error {
	switch s.Action {
	case Mkdir, MkdirCreateOnly:
		return os.MkdirAll(s.Arg0, 0775)

	case Rmdir:
		return os.Remove(s.Arg0)

	case Write, WriteBinary:
		if s.Arg1 != "" {
			return ioutil.WriteFile(s.Arg0, []byte(s.Arg1), 0664)
		} else {
			return nil
		}

	case Remove:
		return os.Remove(s.Arg0)

	case Symlink:
		return os.Symlink(s.Arg0, s.Arg1)

	default:
		return nil
	}
}

func (s Step) PrependPath(path string) (out Step) {
	out = s

	switch s.Action {
	case Noop, Comment:
		return

	case Symlink:
		out.Arg1 = filepath.Join(path, out.Arg1)
	}

	out.Arg0 = filepath.Join(path, out.Arg0)

	return
}

func (s Step) ShellArgs() []string {
	switch s.Action {
	case Noop:
		return nil

	case Comment:
		return []string{"#", s.Arg0}

	case Mkdir, MkdirCreateOnly:
		return []string{"mkdir", "-p", s.Arg0}

	case Rmdir:
		return []string{"rmdir", s.Arg0}

	case Write:
		return []string{"echo", fmt.Sprintf(`"%s"`, s.Arg1), "|", "tee", s.Arg0}

	case WriteBinary:
		var encoded = base64.StdEncoding.EncodeToString([]byte(s.Arg1))
		return []string{"echo", fmt.Sprintf(`"%s"`, encoded), "|", "base64", "-d", "|", "tee", s.Arg0}

	case Remove:
		return []string{"rm", "-f", s.Arg0}

	case Symlink:
		return []string{"ln", "-s", s.Arg0, s.Arg1}

	default:
		return nil
	}
}

func (s Step) Undo() Step {
	switch s.Action {
	case Mkdir:
		return Step{Rmdir, s.Arg0, ""}

	case Symlink:
		return Step{Remove, s.Arg1, ""}

	default:
		return Step{Noop, "", ""}
	}
}

// Steps
type Steps []Step

func (ss *Steps) Append(s Step) Steps {
	*ss = append(*ss, s)
	return *ss
}

func (ss *Steps) Extend(more Steps) Steps {
	*ss = append(*ss, more...)
	return *ss
}

func (steps Steps) Clone() Steps { return slices.Clone(steps) }

func (steps Steps) Reverse() (rev Steps) {
	rev = steps.Clone()
	slices.Reverse(rev)
	return
}

func (steps Steps) Undo() Steps {
	var undo = steps.Clone()

	for i := range undo {
		undo[i] = undo[i].Undo()
	}

	return undo
}

func (steps Steps) PrependPath(path string) Steps {
	for i, s := range steps {
		steps[i] = s.PrependPath(path)
	}

	return steps
}

func (steps Steps) Run() error {
	for i := range steps {
		var s = &steps[i]

		if err := s.Run(); err != nil {
			return fmt.Errorf("step %d %+v: error %w", i, s, err)
		}
	}

	return nil
}

func (steps Steps) ShellArgs() ShellSteps {
	var strs = make(ShellSteps, len(steps))

	for i := range steps {
		strs[i] = steps[i].ShellArgs()
	}

	return strs
}

// ShellSteps
type ShellSteps [][]string

func (shs ShellSteps) Dump(w io.Writer) {
	for _, args := range shs {
		if len(args) == 0 {
			continue
		}

		fmt.Fprintln(w, strings.Join(args, " "))
	}
}
