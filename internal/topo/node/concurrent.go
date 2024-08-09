// Copyright 2024 EMQ Technologies Co., Ltd.
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

package node

import (
	"sync"
	"time"

	"github.com/lf-edge/ekuiper/contract/v2/api"
	"github.com/lf-edge/ekuiper/v2/internal/xsql"
)

// WorkerFunc is the function to process the data
// The function do not need to process error and control messages
// The function must return a slice of data for each input. To omit the data, return nil
type workerFunc func(ctx api.StreamContext, item any) []any

func runWithOrder(ctx api.StreamContext, node *defaultSinkNode, numWorkers int, wf workerFunc) {
	runWithOrderAndInterval(ctx, node, numWorkers, wf, 0)
}

func runWithOrderAndInterval(ctx api.StreamContext, node *defaultSinkNode, numWorkers int, wf workerFunc, sendInterval time.Duration) {
	workerChans := make([]chan any, numWorkers)
	workerOutChans := make([]chan []any, numWorkers)
	for i := range workerChans {
		workerChans[i] = make(chan any)
		workerOutChans[i] = make(chan []any)
	}
	workerExitNotify := make(chan any)
	mergeExitNotify := make(chan any)

	workersWg := &sync.WaitGroup{}

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		workersWg.Add(1)
		go worker(ctx, node, i, wf, workerChans[i], workerOutChans[i], workersWg, workerExitNotify)
	}
	// start merger goroutine
	output := make(chan any)
	go merge(ctx, mergeExitNotify, node, sendInterval, output, workerOutChans...)

	go notifyMergeQuit(workersWg, mergeExitNotify)
	// Distribute input data to workers
	distribute(ctx, node, numWorkers, workerChans, workerExitNotify)
}

// wait all workers exits then notify merge goroutine quit
func notifyMergeQuit(workersWg *sync.WaitGroup, mergeExitNotify chan any) {
	workersWg.Wait()
	close(mergeExitNotify)
}

// Merge multiple channels into one preserving the order
func merge(ctx api.StreamContext, exitNotify chan any, node *defaultSinkNode, sendInterval time.Duration, output chan any, channels ...chan []any) {
	defer close(output)
	// Start a goroutine for each input channel
	for {
		for _, ch := range channels {
			select {
			case data := <-ch:
				for _, d := range data {
					dd, processed := node.commonIngest(ctx, d)
					if processed {
						continue
					}
					node.Broadcast(dd)
					node.statManager.IncTotalRecordsOut()
					if sendInterval > 0 {
						time.Sleep(sendInterval)
					}
				}
			case <-exitNotify:
				ctx.GetLogger().Infof("merge done")
				return
			}
		}
	}
}

func distribute(ctx api.StreamContext, node *defaultSinkNode, numWorkers int, workerChans []chan any, notify chan any) {
	defer func() {
		close(notify)
	}()
	var counter int
	for {
		node.statManager.SetBufferLength(int64(len(node.input)))
		// Round-robin
		if counter == numWorkers {
			counter = 0
		}
		select {
		case <-ctx.Done():
			ctx.GetLogger().Infof("distribute done")
			return
		case item := <-node.input: // Just send out all inputs even they are control tuples
			workerChans[counter] <- item
		}
		counter++
	}
}

func worker(ctx api.StreamContext, node *defaultSinkNode, i int, wf workerFunc, inputRaw chan any, output chan []any, wg *sync.WaitGroup, exitNotify chan any) {
	defer func() {
		wg.Done()
	}()
	for {
		select {
		case data := <-inputRaw:
			item, processed := node.preprocess(ctx, data)
			if processed {
				break
			}
			var result []any
			switch item.(type) {
			case error, *xsql.WatermarkTuple, xsql.EOFTuple:
				result = []any{item}
			default:
				node.statManager.IncTotalRecordsIn()
				result = wf(ctx, item)
				node.statManager.IncTotalMessagesProcessed(1)
			}
			select {
			case output <- result:
			}
		case <-exitNotify:
			ctx.GetLogger().Debugf("worker %d done", i)
			return
		}
	}
}
