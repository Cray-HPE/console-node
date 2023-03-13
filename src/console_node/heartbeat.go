//
//  MIT License
//
//  (C) Copyright 2019-2023 Hewlett Packard Enterprise Development LP
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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type HeartbeatService interface {
	doHeartbeat()
	sendSingleHeartbeat()
	LastHeartbeatTime() string
	SetLastHeartbeatTime(lastHeartbeatTime string)
}

// Include mutex since running in go routine
type HeartbeatManager struct {
	sync.RWMutex
	nodeService        NodeService
	dataService        DataService
	currentNodeService CurrentNodeService
	conmanService      ConmanService
	// Read in via env var
	heartbeatIntervalSecs int
	lastHeartbeatTime     string
}

func NewHeartbeatService(
	ns NodeService,
	ds DataService,
	cns CurrentNodeService,
	cs ConmanService) *HeartbeatManager {

	var heartbeatIntervalSecs int = 30
	readSingleEnvVarInt("HEARTBEAT_SEND_FREQ_SEC", &heartbeatIntervalSecs, 5, 300)

	return &HeartbeatManager{
		nodeService:           ns,
		dataService:           ds,
		currentNodeService:    cns,
		conmanService:         cs,
		heartbeatIntervalSecs: heartbeatIntervalSecs,
		lastHeartbeatTime:     "None",
	}
}

func (hm *HeartbeatManager) LastHeartbeatTime() string {
	hm.RLock()
	defer hm.RUnlock()
	return hm.lastHeartbeatTime
}

func (hm *HeartbeatManager) SetLastHeartbeatTime(lastHeartbeatTime string) {
	hm.Lock()
	hm.lastHeartbeatTime = lastHeartbeatTime
	hm.Unlock()
}

// Function to send heartbeat to console-data
func (hm *HeartbeatManager) doHeartbeat() {
	// NOTE: this is intended to be constantly running in its own thread
	for {
		// do a single heartbeat event
		hm.sendSingleHeartbeat()

		// wait for the next interval
		time.Sleep(time.Duration(hm.heartbeatIntervalSecs) * time.Second)
	}
}

// Function to do the heartbeat
func (hm *HeartbeatManager) sendSingleHeartbeat() {
	// lock the list of current nodes while updating heartbeat information
	currentMtnNodes := hm.currentNodeService.GetMtnNodes().CurrentNodes()
	currentRvrNodes := hm.currentNodeService.GetRvrNodes().CurrentNodes()

	// create the url for the heartbeat of this pod
	url := fmt.Sprintf("%s/consolepod/%s/heartbeat", hm.dataService.DataAddrBase(), podID)

	// gather the current nodes and assemble into json data
	currNodes := make([]nodeConsoleInfo, 0, len(currentMtnNodes)+len(currentRvrNodes))
	for _, ni := range currentRvrNodes {
		currNodes = append(currNodes, *ni)
	}
	for _, ni := range currentMtnNodes {
		currNodes = append(currNodes, *ni)
	}
	data, err := json.Marshal(currNodes)
	if err != nil {
		log.Printf("Error marshalling data for add nodes:%s", err)
		return
	}

	// log last heartbeat time
	t := time.Now()
	hm.SetLastHeartbeatTime(t.Format(time.RFC3339))

	// make the http call
	log.Printf("Pod: %s sending heartbeat", podID)
	rb, _, err := postURL(url, data, nil)
	if err != nil {
		log.Printf("Error sending heartbeat: %s", err)
	}

	// process the nodes no longer controlled by this pod
	if rb != nil {
		// should be an array of nodeConsoleInfo structs
		var droppedNodes []nodeConsoleInfo
		err := json.Unmarshal(rb, &droppedNodes)
		if err != nil {
			log.Printf("Error unmarshalling heartbeat return data: %s", err)
		} else if len(droppedNodes) > 0 {
			log.Printf("Heartbeat: There are %d dropped nodes", len(droppedNodes))

			// release the nodes
			for _, ni := range droppedNodes {
				hm.nodeService.releaseNode(ni.NodeName)
			}

			// signal conman to restart/reconfigure
			hm.conmanService.signalConmanTERM()
		}
	}
}
