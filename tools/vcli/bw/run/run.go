// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package run contains the command allowing to run a sequence of BQL commands
// from the provided file.
package run

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/context"

	"github.com/google/badwolf/bql/grammar"
	"github.com/google/badwolf/bql/planner"
	"github.com/google/badwolf/bql/semantic"
	"github.com/google/badwolf/bql/table"
	"github.com/google/badwolf/storage"
	"github.com/google/badwolf/storage/memory"
	"github.com/google/badwolf/tools/vcli/bw/command"
	"github.com/google/badwolf/tools/vcli/bw/common"
)

// New creates the help command.
func New() *command.Command {
	cmd := &command.Command{
		UsageLine: "run [--channel_size=123] file_path",
		Short:     "runs BQL statements.",
		Long: `Runs all the commands listed in the provided file. Lines in the
the file starting with # will be ignored. All statements will be run
sequentially.
`,
	}
	cmd.Run = func(ctx context.Context, args []string) int {
		return runCommand(ctx, cmd, args)
	}
	return cmd
}

// runCommand runs all the BQL statements available in the file.
func runCommand(ctx context.Context, cmd *command.Command, args []string) int {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Missing required file path. ")
		cmd.Usage()
		return 2
	}
	chanSize := 0
	if len(args) >= 4 {
		c, err := common.ParseChannelSizeFlag(args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fail to parse flag %s with error %v\n", args[2], err)
			return 2
		}
		chanSize = c
	}
	file := strings.TrimSpace(args[len(args)-1])
	lines, err := getStatementsFromFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file %s\n\n\t%v\n\n", file, err)
		return 2
	}
	fmt.Printf("Processing file %s\n\n", args[2])
	s := memory.DefaultStore
	for idx, stm := range lines {
		fmt.Printf("Processing statement (%d/%d):\n%s\n\n", idx+1, len(lines), stm)
		tbl, err := runBQL(ctx, stm, s, chanSize)
		if err != nil {
			fmt.Printf("[FAIL] %v\n\n", err)
			continue
		}
		fmt.Println("Result:")
		if tbl.NumRows() > 0 {
			fmt.Println(tbl)
		}
		fmt.Printf("OK\n\n")
	}
	return 0
}

// runBQL attemps to excecute the provided query against the given store.
func runBQL(ctx context.Context, bql string, s storage.Store, chanSize int) (*table.Table, error) {
	p, err := grammar.NewParser(grammar.SemanticBQL())
	if err != nil {
		return nil, fmt.Errorf("Failed to initilize a valid BQL parser")
	}
	stm := &semantic.Statement{}
	if err := p.Parse(grammar.NewLLk(bql, 1), stm); err != nil {
		return nil, fmt.Errorf("Failed to parse BQL statement with error %v", err)
	}
	pln, err := planner.New(ctx, s, stm, chanSize)
	if err != nil {
		return nil, fmt.Errorf("Should have not failed to create a plan using memory.DefaultStorage for statement %v with error %v", stm, err)
	}
	res, err := pln.Excecute(ctx)
	if err != nil {
		return nil, fmt.Errorf("planner.Execute: failed to execute insert plan with error %v", err)
	}
	return res, nil
}

// getStatementsFromFile returns the statements found in the provided file.
func getStatementsFromFile(path string) ([]string, error) {
	stms, err := common.ReadLines(path)
	if err != nil {
		return nil, err
	}
	return stms, nil
}
