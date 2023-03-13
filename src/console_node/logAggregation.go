//
//  MIT License
//
//  (C) Copyright 2020-2022 Hewlett Packard Enterprise Development LP
//
//  Permission is hereby granted, free of charge, to any person obtaining a
//  copy of this software and associated documentation files (the "Software"),
//  to deal in the Software without restriction, including without limitation
//  the rights to use, copy, modify, merge, publish, distribute, sublicense,
//  and/or sell copies of the Software, and to permit persons to whom the
//  Software is furnished to do so, subject to the following conditions:
//
//  The above copyright notice and this permission notice shall be included
//  in all copies or substantial portions of the Software.
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
//  THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
//  OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
//  ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
//  OTHER DEALINGS IN THE SOFTWARE.
//

// This file contains the code needed to handle aggregation of log files

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hpcloud/tail"
)

type LogAggService interface {
	ConAggLogFile() string
	aggregateFile(xname string) bool
	respinAggLog()
	killTails()
	stopTailing(xname string)
	logPipeOutput(readPipe *io.ReadCloser, desc string)
}

type LogAggManager struct {
	sync.Mutex
	conAggLogger      *log.Logger
	conAggLogFileBase string
	conAggLogFile     string
	tailThreads       map[string]*context.CancelFunc
}

func NewLogAggService() *LogAggManager {
	var conAggLogger *log.Logger = nil
	const conAggLogFileBase string = "/tmp/consoleAgg/consoleAgg-"
	// set the aggregation log name based on the pod name
	var conAggLogFile string = conAggLogFileBase + podName + ".log"
	// map to cancel threads tailing log files
	var tailThreads map[string]*context.CancelFunc = make(map[string]*context.CancelFunc)

	return &LogAggManager{
		conAggLogger:      conAggLogger,
		conAggLogFileBase: conAggLogFileBase,
		conAggLogFile:     conAggLogFile,
		tailThreads:       tailThreads,
	}
}

func (lm *LogAggManager) ConAggLogFile() string {
	return lm.conAggLogFile
}

// Set up tailing a log file to add to the aggregation file
func (lm *LogAggManager) aggregateFile(xname string) bool {
	// NOTE: in update config thread

	newFile := false
	if _, ok := lm.tailThreads[xname]; !ok {
		// indicate we are starting to watch this one
		newFile = true
		// set up a context and a cancel function for this thead
		ctx, cancel := context.WithCancel(context.Background())
		lm.tailThreads[xname] = &cancel

		// record being tracked and forward log file contents
		go lm.watchConsoleLogFile(ctx, xname)
	}
	return newFile
}

// Test function to kill the 'tail' functionality when 'killTails.txt' is created
func (lm *LogAggManager) killTails() {
	for {
		// check if /var/log/console/killTails.txt exists
		if _, err := os.Stat("/var/log/console/killTails.txt"); err == nil {
			// now remove all the tail functions
			for k, tt := range lm.tailThreads {
				log.Printf("Cancelling tail for %s", k)
				(*tt)()
			}

			// empty out the map
			// NOTE - for a true cleanup the entry needs to be removed, but in
			//  debug mode it will just be recreated when conman config is updated.
			//tailThreads = make(map[string]*context.CancelFunc)
			time.Sleep(10 * time.Second)
		} else if os.IsNotExist(err) {
			// file does not exist, so wait and try again later
			time.Sleep(30 * time.Second)
		} else {
			log.Printf("Error looking for killTails.txt file: %s", err)
			return
		}
	}
}

// Function to remove a node from being tailed
func (lm *LogAggManager) stopTailing(xname string) {
	if tt, ok := lm.tailThreads[xname]; ok {
		log.Printf("Halting tail of %s", xname)
		// call the cancel function
		(*tt)()

		// remove from map
		delete(lm.tailThreads, xname)
	} else {
		log.Printf("Stop tailing: could not find %s in tailThreads map", xname)
	}
}

// Watch the input file and append any new content to the aggregate console log file
func (lm *LogAggManager) watchConsoleLogFile(ctx context.Context, xname string) {
	// Keep tailing the input file until the context.Done() is called, then exit

	// Configuration for tail function -
	conf := tail.Config{
		Follow:    true,  // equal to '-F'
		ReOpen:    true,  // if the files is deleted or moved, reopen original file
		MustExist: false, // if file doesn't exist keep trying
		Poll:      true,  // NOTE: it looks like file events don't work - poll instead
		Logger:    tail.DiscardingLogger,
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // set to open at the current end of file
	}

	// full path to the file
	filename := fmt.Sprintf("/var/log/conman/console.%s", xname)
	log.Printf("Starting to parse file: %s", filename)

	// start the tail operation
	tf, err := tail.TailFile(filename, conf)
	if err != nil {
		log.Printf("Failed to tail file %s with error:%s", filename, err)
		return
	}

	// parse the lines of the tail output while looking for a cancel signal
	for {
		select {
		case <-ctx.Done():
			// done tailing this file - exit
			log.Printf("WATCH_CONSOLE: %s exiting gracefully...", xname)

			// received signal to stop so exit gracefully
			// NOTE: unless this is shut down correctly, it will crash when
			//  the next poll interval hits after this removal.
			tf.Config.Poll = false
			tf.Cleanup()
			tf.Stop()
			return
		case line := <-tf.Lines:
			// output the line from the channel
			lm.writeToAggLog(fmt.Sprintf("console.hostname: %s %s", xname, line.Text))
		}
	}
}

// function to manage concurrent writes to the aggregation log
func (lm *LogAggManager) writeToAggLog(str string) {
	lm.Lock()
	defer lm.Unlock()
	if lm.conAggLogger != nil {
		lm.conAggLogger.Printf("%s", str)
	}
}

// Function to close/open a new aggregation logger
func (lm *LogAggManager) respinAggLog() {
	// when the file changes due to log rotation we must recreate the logger
	lm.Lock()
	defer lm.Unlock()

	// make sure the directory exists to put the file in place
	pos := strings.LastIndex(lm.conAggLogFile, "/")
	if pos < 0 {
		log.Printf("Error: console log aggregation file name: %s", lm.conAggLogFile)
		return
	}
	conAggLogDir := lm.conAggLogFile[:pos]
	if _, err := ensureDirPresent(conAggLogDir, 0766); err != nil {
		log.Printf("Failed to respin aggregation file: %s", err)
		return
	}

	log.Printf("Respinning aggregation log")
	calf, err := os.OpenFile(lm.conAggLogFile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Could not open console aggregate log file: %s", err)
	} else {
		log.Printf("Restarted aggregation log file: %s", lm.conAggLogFile)
		lm.conAggLogger = log.New(calf, "", 0)
		lm.conAggLogger.Print("Starting aggregation log")
	}
}

// Take the output of the pipe and log it
func (*LogAggManager) logPipeOutput(readPipe *io.ReadCloser, desc string) {
	log.Printf("Starting log of conmand %s output", desc)
	er := bufio.NewReader(*readPipe)
	for {
		// read the next line
		line, err := er.ReadString('\n')
		if err != nil {
			log.Printf("Ending %s logging from error:%s", desc, err)
			break
		}
		log.Print(line)
	}
}
