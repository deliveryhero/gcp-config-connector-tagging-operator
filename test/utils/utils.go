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

package utils

import (
	"archive/tar"
	"cloud.google.com/go/storage"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/option"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
)

const (
	configConnectorBundleBucket = "configconnector-operator"
	configConnectorVersion      = "1.123.1"
	configConnectorNamespace    = "cnrm-system"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

func RandomSuffix(s string, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("%s-%s", s, string(b))

}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

func InstallConfigConnector(ctx context.Context) error {
	cmd := exec.Command("kubectl", "create", "ns", configConnectorNamespace)
	if _, err := Run(cmd); err != nil {
		return err
	}

	manifestFile, err := os.CreateTemp("", "configconnector-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(manifestFile.Name())

	if err = downloadConfigConnectorManifests(ctx, manifestFile); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "apply", "-f", manifestFile.Name())
	if _, err := Run(cmd); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "apply", "-f", fmt.Sprintf("https://raw.githubusercontent.com/GoogleCloudPlatform/k8s-config-connector/v%s/crds/tags_v1alpha1_tagslocationtagbinding.yaml", configConnectorVersion))
	if _, err := Run(cmd); err != nil {
		return err
	}

	adcPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if adcPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		adcPath = filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
	}

	cmd = exec.Command("kubectl", "create", "secret", "generic", "e2e-gcp-adc-credentials", "--from-file", fmt.Sprintf("key.json=%s", adcPath), "--namespace", configConnectorNamespace)
	if _, err := Run(cmd); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(`
apiVersion: core.cnrm.cloud.google.com/v1beta1
kind: ConfigConnector
metadata:
  name: configconnector.core.cnrm.cloud.google.com
spec:
  mode: cluster
  credentialSecretName: e2e-gcp-adc-credentials
  stateIntoSpec: Absent
---
apiVersion: customize.core.cnrm.cloud.google.com/v1beta1
kind: ControllerResource
metadata:
  name: cnrm-webhook-manager
spec:
  containers:
    - name: webhook
      resources:
        limits:
          memory: 512Mi
        requests:
          memory: 256Mi`)
	if _, err := Run(cmd); err != nil {
		return err
	}

	// wait for operator to be ready
	cmd = exec.Command("kubectl", "wait", "configconnector/configconnector.core.cnrm.cloud.google.com",
		"--for", "jsonpath={.status.healthy}=true",
		"--timeout", "5m",
	)
	if _, err := Run(cmd); err != nil {
		return err
	}
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cnrm-webhook-manager",
		"--for", "condition=Available",
		"--namespace", "cnrm-system",
		"--timeout", "5m",
	)
	if _, err := Run(cmd); err != nil {
		return err
	}

	return nil
}

func UninstallConfigConnector(ctx context.Context) error {
	cmd := exec.Command("kubectl", "delete", "controllerresource", "cnrm-webhook-manager")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	cmd = exec.Command("kubectl", "delete", "configconnector", "configconnector.core.cnrm.cloud.google.com")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	cmd = exec.Command("kubectl", "delete", "secret", "e2e-gcp-adc-credentials", "--namespace", configConnectorNamespace)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	manifestFile, err := os.CreateTemp("", "configconnector-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(manifestFile.Name())

	if err = downloadConfigConnectorManifests(ctx, manifestFile); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "delete", "-f", manifestFile.Name())
	if _, err := Run(cmd); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "delete", "ns", configConnectorNamespace, "--ignore-not-found")
	if _, err := Run(cmd); err != nil {
		return err
	}

	return nil
}

func downloadConfigConnectorManifests(ctx context.Context, file *os.File) error {
	storageClient, err := storage.NewClient(ctx, option.WithoutAuthentication(), storage.WithJSONReads())
	if err != nil {
		return err
	}
	defer storageClient.Close()

	bundleReader, err := storageClient.Bucket(configConnectorBundleBucket).Object(fmt.Sprintf("%s/release-bundle.tar.gz", configConnectorVersion)).NewReader(ctx)
	if err != nil {
		return err
	}
	defer bundleReader.Close()
	gzipReader, err := gzip.NewReader(bundleReader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if hdr.Name == "./operator-system/configconnector-operator.yaml" {
			if _, err := io.Copy(file, tarReader); err != nil {
				return err
			}
			return nil
		}
	}

	return errors.New("no config-connector manifests found in bundle")
}

// LoadImageToKindClusterWithName loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}
