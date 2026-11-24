package repl

import (
	"bufio"
	"fmt"
	"io"
	"mydb/src/executor"
	"mydb/src/sql"
	"strings"
)

type Repl struct {
	executor *executor.Executor
	in       *bufio.Scanner
	out      io.Writer
}

func NewRepl(executor *executor.Executor, in io.Reader, out io.Writer) *Repl {
	return &Repl{
		executor: executor,
		in:       bufio.NewScanner(in),
		out:      out,
	}
}

// run function that keep reading statements from input, until user types .exit
func (r *Repl) Run() error {
	var buf strings.Builder
	fmt.Fprint(r.out, "Welcome to mydb REPL. Type .exit to exit.\n")
	for {
		if buf.Len() == 0 {
			fmt.Fprint(r.out, "mydb> ")
		} else {
			fmt.Fprint(r.out, "		-> ")
		}
		if !r.in.Scan() {
			break
		}
		line := strings.TrimSpace(r.in.Text())
		if line == "" && buf.Len() == 0 {
			continue
		}
		if line == ".exit" && buf.Len() == 0 {
			break
		}
		buf.WriteString(line)
		buf.WriteString(" ")
		if strings.HasSuffix(line, ";") {
			stmt := buf.String()
			buf.Reset()
			err := r.eval(stmt)
			if err != nil {
				fmt.Fprintln(r.out, "ERROR:", err)
			}
		}
	}
	return r.in.Err()
}

// eval function => takes a statement and execute it
func (r *Repl) eval(stmt string) error {
	ast, err := sql.Parse(stmt)
	if err != nil {
		return err
	}
	res, err := r.executor.Execute(ast)
	if err != nil {
		return err
	}

	if len(res.Columns) > 0 {
		for _, row := range res.Rows {
			fmt.Fprintln(r.out, row)
		}
	} else {
		fmt.Fprintln(r.out, res.Message)
	}
	return nil
}
