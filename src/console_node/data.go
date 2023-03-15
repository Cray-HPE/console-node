//
//  MIT License
//
//  (C) Copyright 2021-2022 Hewlett Packard Enterprise Development LP
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

// This file contains the functions to interact with console-data

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type DataService interface {
	acquireNewNodes(numMtn, numRvr int, podLocation *PodLocationDataResponse) []nodeConsoleInfo
	releaseNodes(nodes []nodeConsoleInfo)
	DataAddrBase() string
}

// Implements DataService
type DataManager struct {
	dataAddrBase string
}

// Inject dependencies
func NewDataService() *DataManager {
	return &DataManager{
		dataAddrBase: "http://cray-console-data/v1",
	}
}

func (dm DataManager) DataAddrBase() string {
	return dm.dataAddrBase
}

// Function to acquire new consoles to monitor
func (dm DataManager) acquireNewNodes(numMtn, numRvr int, podLocation *PodLocationDataResponse) []nodeConsoleInfo {
	// NOTE: in doGetNewNodes thread
	log.Printf("Acquiring new nodes mtn: %d, rvr: %d", numMtn, numRvr)
	// TODO: remove after testing
	log.Printf("podLocData xname is %s\n", podLocData.Xname)

	// put together data package
	type ReqData struct {
		NumMtn int    `json:"nummtn"` // Requested number of Mountain nodes
		NumRvr int    `json:"numrvr"` // Requested number of River nodes
		Alias  string `json:"alias"`  // Alias of current node pod is running on
		Xname  string `json:"xname"`  // Xname of current node pod is running on
	}
	data, err := json.Marshal(ReqData{NumMtn: numMtn, NumRvr: numRvr, Alias: podLocation.Alias, Xname: podLocation.Xname})
	if err != nil {
		log.Printf("Error marshalling data:%s", err)
		return nil
	}

	// make the call to console-data
	url := fmt.Sprintf("%s/consolepod/%s/acquire", dm.dataAddrBase, podID)
	rb, _, err := postURL(url, data, nil)
	if err != nil {
		log.Printf("Error in console-data acquire: %s", err)
	}

	// process the return
	var newNodes []nodeConsoleInfo = nil
	if rb != nil {
		// should be an array of nodeConsoleInfo structs
		err := json.Unmarshal(rb, &newNodes)
		if err != nil {
			log.Printf("Error unmarshalling heartbeat return data: %s", err)
		}
	}

	return newNodes
}

// Function to release nodes from this pod
func (dm DataManager) releaseNodes(nodes []nodeConsoleInfo) {
	// NOTE: the current console-data api takes nodeConsoleInfo structs, but really only
	//  needs the xname (as a key).

	// NOTE: calling function needs to protect current nodes lists
	// NOTE: in doGetNewNodes thread
	// NOTE: also called from releaseAllNodes when shutting down

	// create the url for the heartbeat of this pod
	url := fmt.Sprintf("%s/consolepod/%s/release", dm.dataAddrBase, podID)

	// gather the current nodes and assemble into json data
	data, err := json.Marshal(nodes)
	if err != nil {
		log.Printf("Error marshalling data for add nodes:%s", err)
		return
	}

	// make the http call
	log.Printf("Pod: %s releasing nodes", podID)
	_, _, err = postURL(url, data, nil)
	if err != nil {
		log.Printf("Error releasing nodes: %s", err)
	}
}

var debugCtr int = 0

//========================================
// Debugging functions below - not used in production path
//========================================

// NOTE: keeping the below functions for the time being to use when
//  we create a set of integration tests.  They will be moved from
//  here at that time.

/*
func debugNewNodes(numMtn, numRvr int) []nodeConsoleInfo {
	// make 2 fake nodes to return
	var retVal []nodeConsoleInfo = nil

	// create new mountain nodes
	for i := 0; i < numMtn; i++ {
		nn := createTestNI(debugCtr, "Mountain")
		retVal = append(retVal, nn)
		go createTestLogFile(nn.NodeName, false)
		debugCtr++
	}

	// create new river nodes
	for i := 0; i < numRvr; i++ {
		nn := createTestNI(debugCtr, "River")
		retVal = append(retVal, nn)
		go createTestLogFile(nn.NodeName, false)
		debugCtr++
	}

	return retVal
}

// Function to create a fake nodeConsoleInfo based on an id
func createTestNI(id int, cl string) nodeConsoleInfo {
	// put together an xname based on id
	bn := fmt.Sprintf("x1000c1s5b%d", id)
	nn := bn + "n0"
	return nodeConsoleInfo{
		NodeName: nn,
		BmcName:  bn,
		BmcFqdn:  bn,
		Class:    cl,
		NID:      id,
		Role:     "Compute",
	}
}
*/
// DEBUG Function to create and add to a fake log file
func createTestLogFile(xname string, respin bool) {
	// NOTE: this function is only for use in a debug environment where there
	//  are no real console connections present.

	var sleepTime time.Duration = 1 * time.Second
	filename := fmt.Sprintf("/var/log/conman/console.%s", xname)

	// Ff respin is true, only create if the file is not present - meant to
	// be used when a logrotation has moved the original file and we need to
	// create a new one back at the original location.  If the file is still there
	// we do not need to re-create.
	if respin {
		if _, err := os.Stat(filename); err == nil {
			log.Printf("Respinning log file %s, but it exists, so exiting", xname)
			return
		}
	}

	// create and start the log file
	log.Printf("Opening fake log file: %s", filename)
	file1, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("Error creating file: %s", err)
	}
	log1 := log.New(file1, "", log.LstdFlags)

	// start a loop that runs forever to write to the log files
	var lineCnt int64 = 0
	for {
		// write out some bulk
		log1.Print("Start new write:")
		for i := 0; i < 10; i++ {
			log1.Printf("%s, %d: ASAS:LDL:KJFSADSDfDSLKJYUIYHIUNMNKJHSDFKJHDSLKJDFHLKJDSFHASKAJUHSDAASDLKJFHLKJHADSLKJDSHFLKJDHFSD:OUISDFLKDJFHASLJKFHDKJFH", xname, lineCnt)
			lineCnt++
		}

		// wait before writing out again
		time.Sleep(sleepTime)
	}
}
