// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/extensions"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

const processorName = "example-processor"

// a test processor which adds its name to the resource labels and the resource blob.
// the resource blob is expected to be plain text data.
func main() {
	// read the address under which the unix domain socket server should start
	addr := os.Getenv(extensions.ProcessorServerAddressEnv)

	if addr == "" {
		// if addr is not set, use stdin/stdout for communication
		if err := processorRoutine(os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}
	// if addr is set, use unix domain sockets for communication

	h := func(r io.Reader, w io.WriteCloser) {
		if err := processorRoutine(r, w); err != nil {
			log.Fatal(err)
		}
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

func processorRoutine(inputStream io.Reader, outputStream io.WriteCloser) error {
	defer outputStream.Close()

	// split up the input stream into component descriptor, resource, and resource blob
	cd, res, resourceBlobReader, err := utils.ReadProcessorMessage(inputStream)
	if err != nil {
		return err
	}
	if resourceBlobReader != nil {
		defer resourceBlobReader.Close()
	}

	// modify resource blob
	buf := bytes.NewBuffer([]byte{})
	if _, err := io.Copy(buf, resourceBlobReader); err != nil {
		return err
	}
	outputData := fmt.Sprintf("%s\n%s", buf.String(), processorName)

	// modify resource yaml
	l := cdv2.Label{
		Name:  "processor-name",
		Value: json.RawMessage(`"` + processorName + `"`),
	}
	res.Labels = append(res.Labels, l)

	// write modified output to output stream
	if err := utils.WriteProcessorMessage(*cd, res, strings.NewReader(outputData), outputStream); err != nil {
		return err
	}

	return nil
}
