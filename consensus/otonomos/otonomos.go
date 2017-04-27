// Copyright 2017 The go-ethereum Authors
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

package otonomos

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errLargeBlockTime    = errors.New("timestamp too big")
	
)

type Otonomos struct {
	powStartBlock int64 // Consensus engine configuration parameters
	db     ethdb.Database
	pow *ethash.Ethash
	clique *clique.Clique

}

func New(cachedir string, cachesinmem, cachesondisk int, dagdir string, dagsinmem, dagsondisk int, config *params.OtonomosConfig, db ethdb.Database) *Otonomos {
	cliqueConf := &params.CliqueConfig{
		Period: 15,
		Epoch:  30000,
	}
	return  &Otonomos {
		powStartBlock: 360374,
		db:			db,
		pow: 		ethash.New(cachedir, cachesinmem, cachesondisk, dagdir, dagsinmem, dagsondisk),
		clique:	clique.New(cliqueConf, db),
	}
}
// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (otonomos *Otonomos) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (otonomos *Otonomos) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return otonomos.verifyHeader(chain, header, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (otonomos *Otonomos) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := otonomos.verifyHeader(chain, header, seals[i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum ethash engine.
func (otonomos *Otonomos) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	header := block.Header()
	if (header.Number.Int64() < otonomos.powStartBlock) {
		return otonomos.pow.VerifyUncles(chain, block)
	} else {
		return otonomos.clique.VerifyUncles(chain, block)
	}
}

// verifyHeader checks whether a header conforms to the consensus rules
func (otonomos *Otonomos) verifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	if (header.Number.Int64() < otonomos.powStartBlock) {
		return otonomos.pow.VerifyHeader(chain, header, seal)
	} else {
		return otonomos.clique.VerifyHeader(chain, header, seal)
	}
}

// VerifySeal implements consensus.Engine, checking whether the given block satisfies
// the PoW difficulty requirements.
func (otonomos *Otonomos) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	if (header.Number.Int64() < otonomos.powStartBlock) {
		return otonomos.pow.VerifySeal(chain, header)
	} else {
		return otonomos.clique.VerifySeal(chain, header)
	}
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (otonomos *Otonomos) Prepare(chain consensus.ChainReader, header *types.Header) error {
	if (header.Number.Int64() < otonomos.powStartBlock) {
		return otonomos.pow.Prepare(chain, header)
	} else {
		return otonomos.clique.Prepare(chain, header)
	}
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state and assembling the block.
func (otonomos *Otonomos) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	if (header.Number.Int64() < otonomos.powStartBlock) {
		return otonomos.pow.Finalize(chain, header, state, txs, uncles, receipts)
	} else {
		return otonomos.clique.Finalize(chain, header, state, txs, uncles, receipts)
	}
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (otonomos *Otonomos) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()
	if (header.Number.Int64() < otonomos.powStartBlock) {
		return otonomos.pow.Seal(chain, block, stop)
	} else {
		return otonomos.clique.Seal(chain, block, stop)
	}
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (otonomos *Otonomos) APIs(chain consensus.ChainReader) []rpc.API {
	return otonomos.clique.APIs(chain)
}
