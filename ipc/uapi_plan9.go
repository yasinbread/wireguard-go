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
)

// Made up codes for plan9 since I can't use strings
const (
	IpcErrorIO        = 1
	IpcErrorInvalid   = 2
	IpcErrorPortInUse = 3
	IpcErrorUnknown   = 4
	IpcErrorProtocol  = 5
)

const srvPath = "/srv/wg"

func UAPIOpen(name string) (*os.File, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer w.Close()

	srv, err := os.OpenFile(srvPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer srv.Close()

	_, err = fmt.Fprintf(srv, "%d", w.Fd())
	if err != nil {
		return nil, err
	}

	return r, nil
}

type pipeAddr struct{}

func (pipeAddr) Network() string {
	return "pipe"
}

func (pipeAddr) String() string {
	return srvPath
}

type pipeConn struct {
	*os.File
}

func (pipeConn) Write(b []byte) (int, error) {
	return os.Stdout.Write(b)
}

func (pipeConn) Close() error {
	return nil
}

func (pipeConn) LocalAddr() net.Addr {
	return pipeAddr{}
}

func (pipeConn) RemoteAddr() net.Addr {
	return pipeAddr{}
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
