// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package arbtest

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
)

// TestExecutionSpecState runs the test fixtures from execution-spec-tests.
func TestExecutionSpecState(t *testing.T) {
	if !common.FileExist(executionSpecStateTestDir) {
		t.Skipf("directory %s does not exist", executionSpecStateTestDir)
	}
	st := new(testMatcher)

	st.walk(t, executionSpecStateTestDir, func(t *testing.T, name string, test *StateTest) {
		execStateTest(t, st, test)
	})
}

func execStateTest(t *testing.T, st *testMatcher, test *StateTest) {
	for _, subtest := range test.Subtests() {
		key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)

		// If -short flag is used, we don't execute all four permutations, only
		// one.
		executionMask := 0xf
		if testing.Short() {
			executionMask = (1 << (rand.Int63() & 4))
		}
		t.Run(key+"/hash/trie", func(t *testing.T) {
			if executionMask&0x1 == 0 {
				t.Skip("test (randomly) skipped due to short-tag")
			}
			withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
				var result error
				test.Run(subtest, vmconfig, false, rawdb.HashScheme, func(err error, state *StateTestState) {
					result = st.checkFailure(t, err)
				})
				return result
			})
		})
		t.Run(key+"/hash/snap", func(t *testing.T) {
			if executionMask&0x2 == 0 {
				t.Skip("test (randomly) skipped due to short-tag")
			}
			withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
				var result error
				test.Run(subtest, vmconfig, true, rawdb.HashScheme, func(err error, state *StateTestState) {
					if state.Snapshots != nil && state.StateDB != nil {
						if _, err := state.Snapshots.Journal(state.StateDB.IntermediateRoot(false)); err != nil {
							result = err
							return
						}
					}
					result = st.checkFailure(t, err)
				})
				return result
			})
		})
		t.Run(key+"/path/trie", func(t *testing.T) {
			if executionMask&0x4 == 0 {
				t.Skip("test (randomly) skipped due to short-tag")
			}
			withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
				var result error
				test.Run(subtest, vmconfig, false, rawdb.PathScheme, func(err error, state *StateTestState) {
					result = st.checkFailure(t, err)
				})
				return result
			})
		})
		t.Run(key+"/path/snap", func(t *testing.T) {
			if executionMask&0x8 == 0 {
				t.Skip("test (randomly) skipped due to short-tag")
			}
			withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
				var result error
				test.Run(subtest, vmconfig, true, rawdb.PathScheme, func(err error, state *StateTestState) {
					if state.Snapshots != nil && state.StateDB != nil {
						if _, err := state.Snapshots.Journal(state.StateDB.IntermediateRoot(false)); err != nil {
							result = err
							return
						}
					}
					result = st.checkFailure(t, err)
				})
				return result
			})
		})
	}
}

// Transactions with gasLimit above this value will not get a VM trace on failure.
const traceErrorLimit = 400000

func withTrace(t *testing.T, gasLimit uint64, test func(vm.Config) error) {
	// Use config from command line arguments.
	config := vm.Config{}
	err := test(config)
	if err == nil {
		return
	}

	// Test failed, re-run with tracing enabled.
	t.Error(err)
	if gasLimit > traceErrorLimit {
		t.Log("gas limit too high for EVM trace")
		return
	}
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	config.Tracer = logger.NewJSONLogger(&logger.Config{}, w)
	err2 := test(config)
	if !reflect.DeepEqual(err, err2) {
		t.Errorf("different error for second run: %v", err2)
	}
	w.Flush()
	if buf.Len() == 0 {
		t.Log("no EVM operation logs generated")
	} else {
		t.Log("EVM operation log:\n" + buf.String())
	}
	// t.Logf("EVM output: 0x%x", tracer.Output())
	// t.Logf("EVM error: %v", tracer.Error())
}
