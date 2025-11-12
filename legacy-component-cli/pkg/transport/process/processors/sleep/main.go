// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/extensions"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

const sleepTimeEnv = "SLEEP_TIME"

// a test processor which sleeps for a configurable duration and then exits with an error.
func main() {
	sleepTime, err := time.ParseDuration(os.Getenv(sleepTimeEnv))
	if err != nil {
		log.Fatal(err)
	}

	addr := os.Getenv(extensions.ProcessorServerAddressEnv)

	if addr == "" {
		time.Sleep(sleepTime)
		log.Fatal("finished sleeping -> exit with error")
	}

	h := func(r io.Reader, w io.WriteCloser) {
		time.Sleep(sleepTime)
		log.Fatal("finished sleeping -> exit with error")
	}

	srv, err := utils.NewUnixDomainSocketServer(addr, h)
	if err != nil {
		log.Fatal(err)
	}

	srv.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	srv.Stop()
}
