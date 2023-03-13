// MIT License
//
// (C) Copyright 2019-2023 Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.
package main

import "sync"

type NodeConsoleMap interface {
	Put(k string, v *nodeConsoleInfo)
	CurrentNodes() map[string]*nodeConsoleInfo
	ResetCurrentNodes()
}

type MtnNodes struct {
	sync.Mutex
	currNodes map[string]*nodeConsoleInfo
}

func (mn *MtnNodes) Put(k string, v *nodeConsoleInfo) {
	mn.Lock()
	mn.currNodes[k] = v
	mn.Unlock()
}

func (mn *MtnNodes) CurrentNodes() map[string]*nodeConsoleInfo {
	mn.Lock()
	defer mn.Unlock()
	return mn.currNodes
}

func (mn *MtnNodes) ResetCurrentNodes() {
	mn.Lock()
	mn.currNodes = make(map[string]*nodeConsoleInfo)
	mn.Unlock()
}

type RvrNodes struct {
	sync.Mutex
	currNodes map[string]*nodeConsoleInfo
}

func (rn *RvrNodes) Put(k string, v *nodeConsoleInfo) {
	rn.Lock()
	rn.currNodes[k] = v
	rn.Unlock()
}

func (rn *RvrNodes) CurrentNodes() map[string]*nodeConsoleInfo {
	rn.Lock()
	defer rn.Unlock()
	return rn.currNodes
}

func (rn *RvrNodes) ResetCurrentNodes() {
	rn.Lock()
	rn.currNodes = make(map[string]*nodeConsoleInfo)
	rn.Unlock()
}

// Protected locked access to the node maps
type CurrentNodeService interface {
	GetMtnNodes() *MtnNodes
	GetRvrNodes() *RvrNodes
}

type CurrentNodes struct {
	mtnNodes *MtnNodes
	rvrNodes *RvrNodes
}

func (cn *CurrentNodes) GetMtnNodes() *MtnNodes {
	return cn.mtnNodes
}

func (cn *CurrentNodes) GetRvrNodes() *RvrNodes {
	return cn.rvrNodes
}

func NewCurrentNodesService() *CurrentNodes {
	mtnNodes := MtnNodes{currNodes: make(map[string]*nodeConsoleInfo)}
	rvrNodes := RvrNodes{currNodes: make(map[string]*nodeConsoleInfo)}
	return &CurrentNodes{mtnNodes: &mtnNodes, rvrNodes: &rvrNodes}
}
