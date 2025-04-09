//go:build plan9

/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

package ipc

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

// Made up codes for plan9 since I can't use strings
const (
	IpcErrorIO        = 1
	IpcErrorInvalid   = 2
	IpcErrorPortInUse = 3
	IpcErrorUnknown   = 4
	IpcErrorProtocol  = 5
)

const o_RCLOSE = 0x40

func UAPIOpen(name string) (*os.File, error) {
	p := make([]int, 2)
	err := syscall.Pipe(p)
	if err != nil {
		return nil, err
	}
	p0 := os.NewFile(uintptr(p[0]), "p0")
	p1 := os.NewFile(uintptr(p[1]), "p1")
	defer p1.Close()

	path := fmt.Sprintf("/srv/%s", name)
	srv, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|o_RCLOSE, 0666)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(srv, "%d", p1.Fd())
	if err != nil {
		return nil, err
	}

	return p0, nil
}

type pipeAddr struct {
	name string
}

func (pipeAddr) Network() string {
	return "pipe"
}

func (a pipeAddr) String() string {
	return a.name
}

type pipeConn struct {
	*os.File
}

func (pipeConn) Close() error {
	return nil
}

func (c pipeConn) LocalAddr() net.Addr {
	return pipeAddr{name: c.Name()}
}

func (c pipeConn) RemoteAddr() net.Addr {
	return pipeAddr{name: c.Name()}
}

type UAPIListener struct {
	pipe *os.File
}

func (l *UAPIListener) Accept() (net.Conn, error) {
	return pipeConn{l.pipe}, nil
}

func (l *UAPIListener) Close() error {
	return l.pipe.Close()
}

func (l *UAPIListener) Addr() net.Addr {
	return pipeAddr{}
}

func UAPIListen(name string, file *os.File) (net.Listener, error) {
	// wrap file in listener

	uapi := &UAPIListener{pipe: file}

	// TODO: watch for srv deletion

	return uapi, nil
}
