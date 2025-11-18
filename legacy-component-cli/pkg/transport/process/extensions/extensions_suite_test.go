// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package extensions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/extensions"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

const (
	exampleProcessorBinaryPath = "../../../../tmp/test/bin/example-processor"
	sleepProcessorBinaryPath   = "../../../../tmp/test/bin/sleep-processor"
	sleepTimeEnv               = "SLEEP_TIME"
	sleepTime                  = 5 * time.Second
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "transport extensions Test Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	_, err := os.Stat(exampleProcessorBinaryPath)
	Expect(err).ToNot(HaveOccurred(), exampleProcessorBinaryPath+" doesn't exists. pls run make install-requirements.")

	_, err = os.Stat(sleepProcessorBinaryPath)
	Expect(err).ToNot(HaveOccurred(), sleepProcessorBinaryPath+" doesn't exists. pls run make install-requirements.")
}, 5)

var _ = ginkgo.Describe("transport extensions", func() {

	ginkgo.Context("stdio executable", func() {
		ginkgo.It("should create processor successfully if env is nil", func() {
			args := []string{}
			_, err := extensions.NewStdIOExecutable(exampleProcessorBinaryPath, args, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		ginkgo.It("should modify the processed resource correctly", func() {
			args := []string{}
			env := map[string]string{}
			processor, err := extensions.NewStdIOExecutable(exampleProcessorBinaryPath, args, env)
			Expect(err).ToNot(HaveOccurred())

			runExampleResourceTest(processor)
		})

		ginkgo.It("should exit with error when timeout is reached", func() {
			args := []string{}
			env := map[string]string{
				sleepTimeEnv: sleepTime.String(),
			}
			processor, err := extensions.NewStdIOExecutable(sleepProcessorBinaryPath, args, env)
			Expect(err).ToNot(HaveOccurred())

			runTimeoutTest(processor)
		})
	})

	ginkgo.Context("unix domain socket executable", func() {
		ginkgo.It("should create processor successfully if env is nil", func() {
			args := []string{}
			_, err := extensions.NewUnixDomainSocketExecutable(exampleProcessorBinaryPath, args, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		ginkgo.It("should modify the processed resource correctly", func() {
			args := []string{}
			env := map[string]string{}
			processor, err := extensions.NewUnixDomainSocketExecutable(exampleProcessorBinaryPath, args, env)
			Expect(err).ToNot(HaveOccurred())

			runExampleResourceTest(processor)
		})

		ginkgo.It("should raise an error when trying to set the server address env variable manually", func() {
			args := []string{}
			env := map[string]string{
				extensions.ProcessorServerAddressEnv: "/tmp/my-processor.sock",
			}
			_, err := extensions.NewUnixDomainSocketExecutable(exampleProcessorBinaryPath, args, env)
			Expect(err).To(MatchError(fmt.Sprintf("the env variable %s is not allowed to be set manually", extensions.ProcessorServerAddressEnv)))
		})

		ginkgo.It("should exit with error when timeout is reached", func() {
			args := []string{}
			env := map[string]string{
				sleepTimeEnv: sleepTime.String(),
			}
			processor, err := extensions.NewUnixDomainSocketExecutable(sleepProcessorBinaryPath, args, env)
			Expect(err).ToNot(HaveOccurred())

			runTimeoutTest(processor)
		})
	})

})

func runTimeoutTest(processor process.ResourceStreamProcessor) {
	const timeout = 2 * time.Second

	ctx, cancelfunc := context.WithTimeout(context.TODO(), timeout)
	defer cancelfunc()

	err := processor.Process(ctx, bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{}))
	Expect(err).To(MatchError("unable to wait for processor: signal: killed"))
}

func runExampleResourceTest(processor process.ResourceStreamProcessor) {
	const (
		processorName        = "example-processor"
		resourceData         = "12345"
		expectedResourceData = resourceData + "\n" + processorName
	)

	res := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Name:    "my-res",
			Version: "v0.1.0",
			Type:    "ociImage",
		},
	}

	l := cdv2.Label{
		Name:  "processor-name",
		Value: json.RawMessage(`"` + processorName + `"`),
	}
	expectedRes := res
	expectedRes.Labels = append(expectedRes.Labels, l)

	cd := cdv2.ComponentDescriptor{
		ComponentSpec: cdv2.ComponentSpec{
			Resources: []cdv2.Resource{
				res,
			},
		},
	}

	inputBuf := bytes.NewBuffer([]byte{})
	err := utils.WriteProcessorMessage(cd, res, strings.NewReader(resourceData), inputBuf)
	Expect(err).ToNot(HaveOccurred())

	outputBuf := bytes.NewBuffer([]byte{})
	err = processor.Process(context.TODO(), inputBuf, outputBuf)
	Expect(err).ToNot(HaveOccurred())

	processedCD, processedRes, processedBlobReader, err := utils.ReadProcessorMessage(outputBuf)
	Expect(err).ToNot(HaveOccurred())

	Expect(*processedCD).To(Equal(cd))
	Expect(processedRes).To(Equal(expectedRes))

	processedResourceDataBuf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(processedResourceDataBuf, processedBlobReader)
	Expect(err).ToNot(HaveOccurred())

	Expect(processedResourceDataBuf.String()).To(Equal(expectedResourceData))
}
