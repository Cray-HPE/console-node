//
//  MIT License
//
//  (C) Copyright 2023 Hewlett Packard Enterprise Development LP
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

// This file contains the functions to monitor for changes in keys and certs

package main

import (
	"bytes"
	"crypto/sha256"
	"io"
	"log"
	"os"
	"time"

	compcreds "github.com/Cray-HPE/hms-compcredentials"
)

// Time to wait between checking for credential changes
var monitorIntervalSecs int = 30

var previousPrivateKeyHash []byte = nil
var previousPublicKeyHash []byte = nil

var previousPasswords map[string]compcreds.CompCredentials = nil

// function to do check for credential changes and restart conman if necessary
func checkForChanges(conmanService ConmanService, credService CredService) {
	restartConman := false

	// check for changes in the mountain key files
	if checkIfMountainConsoleKeysChanged(credService) {
		restartConman = true
	}

	// check for changes in river keys
	if checkIfRiverPasswordsChanged(credService) {
		// the config file will be updated in the runConman thread when conman is restarted
		restartConman = true
	}

	//restart conman if necessary
	if restartConman {
		conmanService.signalConmanTERM()
	}
}

// function to continuously monitor for changes that require conman to restart
func doMonitor(conmanService ConmanService, credService CredService) {
	// NOTE: this is intended to be constantly running in its own thread
	for {
		// do a single monitor event
		checkForChanges(conmanService, credService)

		// wait for the next interval
		time.Sleep(time.Duration(monitorIntervalSecs) * time.Second)
	}
}

// function to check if the passwords have changed since conman was configured
func checkIfRiverPasswordsChanged(credService CredService) bool {
	if previousPasswords == nil {
		// this shouldn't happen due to the order of initilization, but just to be safe we skip this case.
		return false
	}

	currNodesMutex.Lock()
	defer currNodesMutex.Unlock()

	var rvrXNames []string = nil
	for _, nodeCi := range currentRvrNodes {
		rvrXNames = append(rvrXNames, nodeCi.BmcName)
	}
	// don't retry here so we don't block heartbeats with the mutex.  we can check again the next pass
	currentPasswords := credService.getPasswords(rvrXNames)

	for _, nodeCi := range currentRvrNodes {
		currentCreds, ok := currentPasswords[nodeCi.BmcName]
		if !ok {
			log.Printf("Missing credentials detected for %s while checking for credential changes", nodeCi.BmcName)
			continue
		}
		previousCreds, _ := previousPasswords[nodeCi.BmcName]
		if (currentCreds.Username != previousCreds.Username) || (currentCreds.Password != previousCreds.Password) {
			log.Printf("Change detected in the river passwords.  Conman will be reconfigured.")
			return true
		}
	}
	return false
}

// function to check if the console keys have changed since the last run of this function
func checkIfMountainConsoleKeysChanged(credService CredService) bool {
	var keysChanged bool = false

	if len(currentMtnNodes) == 0 {
		// if no mountain nodes are monitored, the keys don't matter
		return false
	}

	// load hashes of both the public and private key files for comparison
	currentPrivateKeyHash, err := hashFile(credService.MountainConsoleKey())
	if err != nil {
		log.Printf("Error generating a hash of the private console key: %s", err)
		return false
	}
	currentPublicKeyHash, err := hashFile(credService.MountainConsoleKeyPub())
	if err != nil {
		log.Printf("Error generating a hash of the public console key: %s", err)
		return false
	}

	// don't register a change if this is the first time and the fields are empty
	if previousPrivateKeyHash != nil && previousPublicKeyHash != nil {
		// if one key changes the other should change, but this checks both for safety
		if !(bytes.Equal(currentPrivateKeyHash, previousPrivateKeyHash)) {
			keysChanged = true
		}
		if !(bytes.Equal(currentPublicKeyHash, previousPublicKeyHash)) {
			keysChanged = true
		}
	}

	previousPrivateKeyHash = currentPrivateKeyHash
	previousPublicKeyHash = currentPublicKeyHash

	if keysChanged {
		log.Printf("Change detected in the mountain keys.  Conman will be restarted.")
	}
	return keysChanged
}

// returns a hash of the given file
func hashFile(fileName string) ([]byte, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}
