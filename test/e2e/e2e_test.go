/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"google.golang.org/api/option"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/deliveryhero/gcp-config-connector-tagging-operator/test/utils"
)

const (
	namespace = "gcp-config-connector-tagging-operator-system"
)

var _ = Describe("controller", Ordered, func() {
	ctx := context.Background()

	// We avoid the use of AfterAll due to ordering issues: https://github.com/onsi/ginkgo/issues/1284#issuecomment-1756314394
	BeforeAll(func() {
		By("installing config connector")
		Expect(utils.InstallConfigConnector(ctx)).To(Succeed())
		DeferCleanup(func() {
			By("uninstalling the config connector bundle")
			Expect(utils.UninstallConfigConnector(ctx)).To(Succeed())
		})

		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, _ = utils.Run(cmd)
		DeferCleanup(func() {
			By("removing manager namespace")
			cmd := exec.Command("kubectl", "delete", "ns", namespace)
			_, _ = utils.Run(cmd)
		})
	})

	Context("Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error

			// projectimage stores the name of the image used in the example
			var projectimage = "ghcr.io/deliveryhero/gcp-config-connector-tagging-operator:e2e"

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName(projectimage)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name

				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())
		})

		It("should apply tags to resources", func() {
			projectID := os.Getenv("GCP_PROJECT")
			Expect(projectID).NotTo(BeEmpty(), "Environment variable GCP_PROJECT must be set")
			location := os.Getenv("GCP_LOCATION")
			if location == "" {
				location = "europe-west1"
			}
			bindingsClient, err := resourcemanager.NewTagBindingsClient(ctx, option.WithEndpoint(location+"-cloudresourcemanager.googleapis.com"))
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			valuesClient, err := resourcemanager.NewTagValuesClient(ctx)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("creating a namespace")
			resourceNamespace := utils.RandomSuffix("test-resources", 5)
			cmd := exec.Command("kubectl", "create", "ns", resourceNamespace)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				cmd := exec.Command("kubectl", "delete", "ns", resourceNamespace)
				_, err := utils.Run(cmd)
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
			})
			cmd = exec.Command("kubectl", "annotate", "ns", resourceNamespace, fmt.Sprintf("cnrm.cloud.google.com/project-id=%s", projectID))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that a storage bucket is tagged")
			bucketName := utils.RandomSuffix(fmt.Sprintf("%s-e2e", projectID), 5)
			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(fmt.Sprintf(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  labels:
    foo: bar
	key1: value1
	key2: value2
  name: %s
  namespace: %s
spec:
  location: %s
  publicAccessPrevention: enforced
  uniformBucketLevelAccess: true`, bucketName, resourceNamespace, location))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, waitForResourceReady("storagebucket", bucketName, resourceNamespace)).NotTo(HaveOccurred())
			EventuallyWithOffset(1, verifyHasTagValue(ctx, bindingsClient, valuesClient, fmt.Sprintf("//storage.googleapis.com/projects/_/buckets/%s", bucketName), namespacedTagValue(projectID, "foo/bar")), time.Minute, time.Second).Should(Succeed())
		})
	})
})

func verifyHasTagValue(ctx context.Context, bindingsClient *resourcemanager.TagBindingsClient, valuesClient *resourcemanager.TagValuesClient, parent string, tagValue string) func(Gomega) {
	return func(g Gomega) {
		tagValues, err := utils.GetResourceTagValues(ctx, bindingsClient, valuesClient, parent)
		g.ExpectWithOffset(2, err).NotTo(HaveOccurred())
		g.ExpectWithOffset(2, len(tagValues)).To(BeNumerically(">=", 1), "Expected at least one tag value to be bound")
		g.ExpectWithOffset(2, tagValues).To(ContainElement(tagValue))
	}
}

func namespacedTagValue(projectID string, tagValue string) string {
	return fmt.Sprintf("%s/%s", projectID, tagValue)
}

func waitForResourceReady(resourceType string, resource string, namespace string) error {
	cmd := exec.Command("kubectl", "wait", fmt.Sprintf("%s/%s", resourceType, resource),
		"--for", "condition=Ready",
		"--namespace", namespace,
		"--timeout", "5m",
	)
	_, err := utils.Run(cmd)
	return err
}
