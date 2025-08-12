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
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestExecutionSpecBlocktests runs the test fixtures from execution-spec-tests.
func TestExecutionSpecBlocktests(t *testing.T) {
	if !common.FileExist(executionSpecBlockchainTestDir) {
		t.Skipf("directory %s does not exist", executionSpecBlockchainTestDir)
	}
	bt := new(testMatcher)

	// These tests fail as of https://github.com/ethereum/go-ethereum/pull/28666, since we
	// no longer delete "leftover storage" when deploying a contract.
	bt.skipLoad(`^cancun/eip6780_selfdestruct/selfdestruct/self_destructing_initcode_create_tx.json`)
	bt.skipLoad(`^cancun/eip6780_selfdestruct/selfdestruct/self_destructing_initcode.json`)
	bt.skipLoad(`^cancun/eip6780_selfdestruct/selfdestruct/recreate_self_destructed_contract_different_txs.json`)
	bt.skipLoad(`^cancun/eip6780_selfdestruct/selfdestruct/delegatecall_from_new_contract_to_pre_existing_contract.json`)

	// On Arbitrum ModExp broken on Paris and Cancun, but not Shanghai?
	bt.skipLoad(`test_modexp\.py::test_modexp\[fork_(Cancun|Paris)`)

	// Not all opcodes supported on Mainnet are supported on Arbitrum.
	bt.skipLoad(`test_all_opcodes\.py::test_all_opcodes\[fork_(Cancun|Paris)`)

	// Arbitrum doesn't support withdrawals
	bt.skipLoad(`^shanghai/eip4895_withdrawals/withdrawals/.*\.json$`)

	// Arbitrum doesn't support eip4788 beacon root
	bt.skipLoad(`^cancun/eip4788_beacon_root/.*\.json$`)

	// Arbitrum doesn't support eip4844
	bt.skipLoad(`^cancun/eip4844_blobs/.+\.json$`)
	bt.skipLoad(`^cancun/eip7516_blobgasfee/.+\.json$`)

	bt.walk(t, executionSpecBlockchainTestDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test)
	})
}

func execBlockTest(t *testing.T, bt *testMatcher, test *BlockTest) {
	// Define all the different flag combinations we should run the tests with,
	// picking only one for short tests.
	//
	// Note, witness building and self-testing is always enabled as it's a very
	// good test to ensure that we don't break it.
	var (
		snapshotConf = []bool{false, true}
		dbschemeConf = []string{rawdb.HashScheme, rawdb.PathScheme}
	)
	if testing.Short() {
		snapshotConf = []bool{snapshotConf[rand.Int()%2]}
		dbschemeConf = []string{dbschemeConf[rand.Int()%2]}
	}
	for _, snapshot := range snapshotConf {
		for _, dbscheme := range dbschemeConf {
			fmt.Println("Running test with config", snapshot, dbschemeConf)
			if err := bt.checkFailure(t, test.Run(snapshot, dbscheme, false, nil, nil)); err != nil {
				t.Errorf("test with config {snapshotter:%v, scheme:%v} failed: %v", snapshot, dbscheme, err)
				return
			}
		}
	}
}
