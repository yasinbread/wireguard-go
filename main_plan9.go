//go:build plan9

/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	//	"strconv"

	"github.com/tailscale/wireguard-go/conn"
	"github.com/tailscale/wireguard-go/device"
	"github.com/tailscale/wireguard-go/ipc"
	"github.com/tailscale/wireguard-go/tun"
	"golang.org/x/sys/plan9"
)

const (
	ExitSetupSuccess = 0
	ExitSetupFailed  = 1
)

const (
	ENV_WG_TUN_FD  = "WG_TUN_FD"
	ENV_WG_UAPI_FD = "WG_UAPI_FD"
)

func printUsage() {
	fmt.Printf("Usage: %s INTERFACE-NAME\n", os.Args[0])
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("wireguard-go v%s\n\nUserspace WireGuard daemon for %s-%s.\nInformation available at https://www.wireguard.com.\nCopyright (C) Jason A. Donenfeld <Jason@zx2c4.com>.\n", Version, runtime.GOOS, runtime.GOARCH)
		return
	}

	var interfaceName string

	if len(os.Args) != 2 {
		printUsage()
		return
	}
	interfaceName = os.Args[1]

	// get log level (default: info)

	logLevel := func() int {
		switch os.Getenv("LOG_LEVEL") {
		case "verbose", "debug":
			return device.LogLevelVerbose
		case "error":
			return device.LogLevelError
		case "silent":
			return device.LogLevelSilent
		}
		return device.LogLevelError
	}()

	// open TUN device

	tdev, err := tun.CreateTUN(interfaceName, device.DefaultMTU)

	if err == nil {
		realInterfaceName, err2 := tdev.Name()
		if err2 == nil {
			interfaceName = realInterfaceName
		}
	}

	logger := device.NewLogger(
		logLevel,
		fmt.Sprintf("(%s) ", interfaceName),
	)

	logger.Verbosef("Starting wireguard-go version %s", Version)

	if err != nil {
		logger.Errorf("Failed to create TUN device: %v", err)
		os.Exit(ExitSetupFailed)
	}

	// open UAPI file (or use supplied fd)

	fileUAPI, err := ipc.UAPIOpen(interfaceName)
	if err != nil {
		logger.Errorf("UAPI listen error: %v", err)
		os.Exit(ExitSetupFailed)
		return
	}
	defer fileUAPI.Close()

	// TODO: daemonize

	device := device.NewDevice(tdev, conn.NewDefaultBind(), logger)

	logger.Verbosef("Device started")

	errs := make(chan error)
	term := make(chan os.Signal, 1)

	uapi, err := ipc.UAPIListen(interfaceName, fileUAPI)
	if err != nil {
		logger.Errorf("Failed to listen on uapi socket: %v", err)
		os.Exit(ExitSetupFailed)
	}

	go func() {
		for {
			conn, err := uapi.Accept()
			if err != nil {
				errs <- err
				return
			}

			device.IpcHandle(conn)
		}
	}()

	logger.Verbosef("UAPI listener started")

	// wait for program to terminate

	signal.Notify(term, plan9.Note("interrupt"))
	signal.Notify(term, os.Interrupt)

	select {
	case <-term:
	case <-errs:
	case <-device.Wait():
	}

	// clean up

	uapi.Close()
	device.Close()

	logger.Verbosef("Shutting down")
}
