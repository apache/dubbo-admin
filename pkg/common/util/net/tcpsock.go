/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package net

import (
	"fmt"
	"net"
)

func PickTCPPort(ip string, leftPort, rightPort uint32) (uint32, error) {
	lowestPort, highestPort := leftPort, rightPort
	if highestPort < lowestPort {
		lowestPort, highestPort = highestPort, lowestPort
	}
	// we prefer a port to remain stable over time, that's why we do sequential availability check
	// instead of random selection
	for port := lowestPort; port <= highestPort; port++ {
		if actualPort, err := ReserveTCPAddr(fmt.Sprintf("%s:%d", ip, port)); err == nil {
			return actualPort, nil
		}
	}
	return 0, fmt.Errorf("unable to find port in range %d:%d", lowestPort, highestPort)
}

func ReserveTCPAddr(address string) (uint32, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}
