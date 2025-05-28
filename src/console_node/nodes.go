//
//  MIT License
//
//  (C) Copyright 2019-2025 Hewlett Packard Enterprise Development LP
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

// This file contains the code to manage the node consoles under this pod

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Struct to hold all node level information needed to form a console connection
type nodeConsoleInfo struct {
	NodeName string // node xname
	BmcName  string // bmc xname
	BmcFqdn  string // full name of bmc
	Class    string // river/mtn class
	NID      int    // NID of the node
	Role     string // role of the node
}

// Function to determine if a node is Mountain hardware
func (node nodeConsoleInfo) isMountain() bool {
	return node.Class == "Mountain" || node.Class == "Hill"
}

// Function to determine if a node is River hardware
func (node nodeConsoleInfo) isRiver() bool {
	return node.Class == "River"
}

// Function to determine if a node is Paradise hardware
func (node nodeConsoleInfo) isParadise() bool {
	return node.Class == "Paradise"
}

// Provide a function to convert struct to string
func (node nodeConsoleInfo) String() string {
	return fmt.Sprintf("NodeName:%s, BmcName:%s, BmcFqdn:%s, Class:%s, NID:%d, Role:%s",
		node.NodeName, node.BmcName, node.BmcFqdn, node.Class, node.NID, node.Role)
}

// Globals for managing nodes being watched
// NOTE: the current nodes need to be kept in 3 distinct groups:
//
//	River Nodes: connect through ipmi protocol directly through conman
//	Mountain Nodes: connect through expect script via passwordless ssh
//	Paradise Nodes: connect through expect script via password based ssh
var currNodesMutex = &sync.Mutex{}
var currentMtnNodes map[string]*nodeConsoleInfo = make(map[string]*nodeConsoleInfo) // [xname,*consoleInfo]
var currentRvrNodes map[string]*nodeConsoleInfo = make(map[string]*nodeConsoleInfo) // [xname,*consoleInfo]
var currentPdsNodes map[string]*nodeConsoleInfo = make(map[string]*nodeConsoleInfo) // [xname,*consoleInfo]

// Number of nodes this pod should be watching
// NOTE: deprecated, prefer to call OperatorService.getCurrentTargets()
var targetRvrNodes int = -1
var targetMtnNodes int = -1

// Number of nodes to get per acquisition query
var maxAcquireRvr int = 500
var maxAcquireMtn int = 200

// Pause between each lookup for new node information
var newNodeLookupSec int = 30

// File to hold target number of node information - it will reside on
// a shared file system so console-node pods can read what is set here
const targetNodeFile string = "/var/log/console/TargetNodes.txt"

// function to safely get the current node xnames
func getCurrNodeXnames() []string {
	// put a lock on the current nodes while looking for new ones
	currNodesMutex.Lock()
	log.Print(("getCurrNodeXnames:: locking mutex"))
	defer func() {
		currNodesMutex.Unlock()
		log.Print("getCurrNodeXnames:: unlocking mutex")
	}()

	// gather the names of all the current nodes being watched
	var retVal []string
	for key := range currentMtnNodes {
		retVal = append(retVal, key)
	}
	for key := range currentRvrNodes {
		retVal = append(retVal, key)
	}
	for key := range currentPdsNodes {
		retVal = append(retVal, key)
	}

	return retVal
}

// function to see if a node is being monitored
func isNodeMonitored(xname string) bool {
	// put a lock on the current nodes while looking for new ones
	currNodesMutex.Lock()
	log.Print(("isNodeMonitored:: locking mutex"))
	defer func() {
		currNodesMutex.Unlock()
		log.Print("isNodeMonitored:: unlocking mutex")
	}()

	// check if this node is being monitored
	if _, ok := currentMtnNodes[xname]; ok {
		return true
	} else if _, ok := currentRvrNodes[xname]; ok {
		return true
	} else if _, ok := currentPdsNodes[xname]; ok {
		return true
	}
	return false
}

// small helper function to insure correct number of nodes asked for
func pinNumNodes(numAsk, numMax int) int {
	// insure the input number ends in range [0,numMax]
	if numAsk < 0 {
		// already have too many
		numAsk = 0
	} else if numAsk > numMax {
		// pin at the maximum
		numAsk = numMax
	}
	return numAsk
}

// local max function for ints
func locMax(a, b int) int {
	// NOTE: this may be removed in favor of the standard 'max' function when we upgrade past 1.18
	if a > b {
		return a
	}
	return b
}

// helper function to calculate how many nodes to ask for
func calcChangeInNodes() (deltaMtn, deltaRvr int) {
	// The change to the number of nodes being monitored - positive
	// means to acquire more, negative to release some.
	deltaMtn = 0
	deltaRvr = 0

	// Update the target number of nodes being monitored
	updateNodesPerPod()

	// NOTE: the 'target' values should be the total number of nodes divided
	//  by the number of console-node pods. Where this gets tricky is if
	//  one or more console-node pods has failed. Then we want to take over
	//  the unmonitored nodes until that pod is back up, then give them back.
	//  We also always want to have extra space to allow a node to shift so
	//  a console-node pod isn't monitoring the worker it is running on.

	// Get the current number of nodes being monitored here
	currNumRvr := len(currentRvrNodes)
	currNumMtn := len(currentMtnNodes)

	// get the number of currently active nodes from the data service
	numPods, errNumPods := getNumActiveNodePods()
	if errNumPods != nil {
		log.Print("Unable to find current number of active nodes, defaulting to 1")
		numPods = 1
	}

	// get the current targets from the operator service, default to something
	// reasonable if we can't contact the operator service
	currTargets, errCurrTargets := opService.getCurrentTargets()
	if errCurrTargets != nil {
		log.Print("Unable to find current targets from operator service - defaulting to something reasonable")
	}

	// Figure out how to adjust the number of nodes managed by this pod
	if errCurrTargets == nil && errNumPods == nil {
		// we have good current information from data and operator services, so use it
		// Try to balance the number of nodes with the total of node pods
		safeNumPods := locMax(numPods, 1)
		idealNumRvr := (currTargets.TotalRvrNodes / safeNumPods) + 1
		idealNumMtn := (currTargets.TotalMtnNodes / safeNumPods) + 1

		// if we are under the ideal number, try to match
		// if we are more than 10% over the ideal, give some back
		// NOTE: there is a gap where we don't change the number to try and avoid bouncing
		if (currNumRvr < idealNumRvr) || (float64(currNumRvr) > (1.1 * float64(idealNumRvr))) {
			// simple difference from ideal - could be positive or negative
			deltaRvr = idealNumRvr - currNumRvr
		}
		if (currNumMtn < idealNumMtn) || (float64(currNumMtn) > (1.1 * float64(idealNumMtn))) {
			// simple difference from ideal - could be positive or negative
			deltaMtn = idealNumMtn - currNumMtn
		}
	} else {
		log.Printf("calcChangeInNodes: unable to do detailed calculation")
		// we have having problems contacting the data and operator services, so guess
		deltaRvr = pinNumNodes(targetRvrNodes-currNumRvr, maxAcquireRvr)
		deltaMtn = pinNumNodes(targetMtnNodes-currNumMtn, maxAcquireMtn)
	}

	return deltaMtn, deltaRvr
}

func doGetNewNodes() {
	// if the pod is shutting down, don't touch the current nodes
	if inShutdown {
		log.Print("In pod shutdown, skipping doGetNewNodes")
		return
	}

	// put a lock on the current nodes while looking for new ones
	currNodesMutex.Lock()
	log.Print(("doGetNewNodes:: locking mutex"))
	defer func() {
		currNodesMutex.Unlock()
		log.Print("doGetNewNodes:: unlocking mutex")
	}()

	// keep track of if we need to redo the configuration
	changed := false

	// Figure out how to adjust the number of nodes being monitored
	deltaMtn, deltaRvr := calcChangeInNodes()

	// Make sure we can always take one river node if we need to move a worker node
	if deltaRvr == 0 {
		deltaRvr = 1
	}

	log.Printf("doGetNewNodes - deltaRvr: %d, deltaMtn: %d", deltaRvr, deltaMtn)

	// From the change numbers, pull out how many to add (if any)
	// NOTE: paradise nodes are included in mountain count
	numAcqRvr := pinNumNodes(deltaRvr, maxAcquireRvr)
	numAcqMtn := pinNumNodes(deltaMtn, maxAcquireMtn)

	if numAcqRvr > 0 || numAcqMtn > 0 {
		newNodes := acquireNewNodes(numAcqMtn, numAcqRvr, podLocData)
		// process the new nodes
		// NOTE: this should be the ONLY place where the maps of
		//  current nodes is updated!!!
		newRvr := 0
		newMtn := 0
		newPds := 0
		for i, node := range newNodes {
			if node.isRiver() {
				currentRvrNodes[node.NodeName] = &newNodes[i]
				changed = true
				newRvr++
			} else if node.isMountain() {
				currentMtnNodes[node.NodeName] = &newNodes[i]
				changed = true
				newMtn++
			} else if node.isParadise() {
				currentPdsNodes[node.NodeName] = &newNodes[i]
				changed = true
				newPds++
			}
		}
		log.Printf(" Added River:%d, Mountain:%d, Paradise:%d", newRvr, newMtn, newPds)
	}

	// See if we have too many nodes
	if rebalanceNodes(deltaRvr, deltaMtn) {
		changed = true
	}

	// Restart the conman process if needed
	if changed {
		// trigger a re-configuration and restart of conman
		signalConmanTERM()

		// rebuild the log rotation configuration file
		updateLogRotateConf()
	}

}

// Primary loop to watch for updates
func watchForNodes() {
	// create a loop to execute the conmand command
	for {
		// look for new nodes once
		doGetNewNodes()

		// Wait for the correct polling interval
		time.Sleep(time.Duration(newNodeLookupSec) * time.Second)
	}
}

// If we have too many nodes, release some
func rebalanceNodes(deltaRvr, deltaMtn int) bool {
	// NOTE: this function just modifies currentNodes lists and stops
	//  tailing operation.  The configuration files will be triggered to be
	//  regenerated outside of this operation.

	// NOTE: in doGetNewNodes thread

	// gather nodes to give back
	var rn []nodeConsoleInfo

	// release river nodes until match target number
	// NOTE: map iteration is random
	if deltaRvr < 0 {
		endNumRvr := len(currentRvrNodes) + deltaRvr
		for key, ni := range currentRvrNodes {
			if len(currentRvrNodes) > endNumRvr {
				// remove another one
				rn = append(rn, *ni)
				delete(currentRvrNodes, key)

				// stop tailing this file
				stopTailing(key)
			} else {
				// done so break
				break
			}
		}
	}

	// release mtn nodes until match target number
	// NOTE: paradise nodes count towards mountain limits, remove from both
	if deltaMtn < 0 {
		endNumMtn := len(currentMtnNodes) + len(currentPdsNodes) + deltaMtn
		for len(currentMtnNodes)+len(currentPdsNodes) > endNumMtn {
			// balance removal so take from whichever pool is larger, one at a time
			targetPool := &currentPdsNodes
			if len(currentMtnNodes) > len(currentPdsNodes) {
				targetPool = &currentMtnNodes
			}

			// make sure we didn't hit some weird condition where both lists are empty
			if len(*targetPool) == 0 {
				break
			}

			// remove a node from the target pool
			// NOTE: map iteration is random - use it to grab a random node to remove
			for key, ni := range *targetPool {
				// remove node
				rn = append(rn, *ni)
				delete(*targetPool, key)

				// stop tailing this file
				stopTailing(key)

				// only want to remove one at a time
				break
			}
		}
	}

	if len(rn) > 0 {
		log.Printf("Rebalance operation is releasing %d nodes", len(rn))
		// notify console-data that we are no longer tracking these nodes
		releaseNodes(rn)

		// signify that we have removed nodes and something has changed
		return true
	}

	// signify nothing has really changed
	return false
}

// Function to release the node from being monitored
func releaseNode(xname string) bool {
	// NOTE: called during heartbeat thread

	// This will remove it from the list of current nodes and stop tailing the
	// log file.
	found := false
	if _, ok := currentRvrNodes[xname]; ok {
		delete(currentRvrNodes, xname)
		found = true
	} else if _, ok := currentMtnNodes[xname]; ok {
		delete(currentMtnNodes, xname)
		found = true
	} else if _, ok := currentPdsNodes[xname]; ok {
		delete(currentPdsNodes, xname)
		found = true
	}

	// remove the tail process for this file
	stopTailing(xname)

	return found
}

// Update the number of target consoles per node pod
func updateNodesPerPod() {

	// NOTE: this is in the process of being deprecated - now the number of
	//  targeted nodes should be retrieved through the console-operator
	//  http api via the OperatorService.getCurrentTargets() function call.
	//  Ths is being left in for a backup mechanism in case the http function
	//  fails.

	// NOTE: in doGetNewNodes thread

	log.Printf("Updating nodes per pod")
	// open the state file
	sf, err := os.Open(targetNodeFile)
	if err != nil {
		log.Printf("Unable to open target node file %s: %s", targetNodeFile, err)
		return
	}
	defer sf.Close()

	// process the lines in the file
	newRvr := -1
	newMtn := -1
	er := bufio.NewReader(sf)
	for {
		// read the next line
		line, err := er.ReadString('\n')
		if err != nil {
			// done reading file
			break
		}

		// find if this is a river line
		const rvrTxt string = "River:"
		const mtnTxt string = "Mountain:"

		if pos := strings.Index(line, rvrTxt); pos >= 0 {
			// peel out the number between : and eol
			numStr := line[pos+len(rvrTxt) : len(line)-1]
			newRvr, err = strconv.Atoi(numStr)
			if err != nil {
				log.Printf("Error reading number of river nodes: %s", err)
			}
		}

		// find if this is a mountain line
		if pos := strings.Index(line, mtnTxt); pos >= 0 {
			// peel out the number between : and eol
			numStr := line[pos+len(mtnTxt) : len(line)-1]
			newMtn, err = strconv.Atoi(numStr)
			if err != nil {
				log.Printf("Error reading number of mountain nodes: %s", err)
			}
		}
	}

	// set the new values with a little sanity checking
	if newRvr >= 0 {
		targetRvrNodes = newRvr
	}
	if newMtn >= 0 {
		targetMtnNodes = newMtn
	}
	log.Printf("  New target nodes - mtn: %d, rvr: %d", newMtn, newRvr)
}
