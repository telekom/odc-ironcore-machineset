/*
Copyright 2026 Deutsche Telekom AG.

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

package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ironcore-dev/controller-utils/buildutils"
	"github.com/ironcore-dev/controller-utils/modutils"
	ironcorecomputev1alpha1 "github.com/ironcore-dev/ironcore/api/compute/v1alpha1"
	ipamv1alpha1 "github.com/ironcore-dev/ironcore/api/ipam/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	storagev1alpha1 "github.com/ironcore-dev/ironcore/api/storage/v1alpha1"
	utilsenvtest "github.com/ironcore-dev/ironcore/utils/envtest"
	"github.com/ironcore-dev/ironcore/utils/envtest/apiserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	computev1alpha1 "github.com/telekom/odc-ironcore-machineset/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// apiServiceTimeout is how long to wait for IronCore's API server. The first run builds it from
// source, which can take a few minutes.
const apiServiceTimeout = 5 * time.Minute

var (
	ctx        context.Context
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	testEnvExt *utilsenvtest.EnvironmentExtensions
	cfg        *rest.Config
	k8sClient  client.Client
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

// The test environment is a real Kubernetes API server and etcd (envtest) with our CRDs, plus
// IronCore's own API server. IronCore serves its types (Machine, Volume, ...) through an
// aggregated API server: a separate server that the Kubernetes API server forwards requests to.
var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	Expect(computev1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(ironcorecomputev1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(storagev1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(networkingv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(ipamv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
	testEnvExt = &utilsenvtest.EnvironmentExtensions{
		APIServiceDirectoryPaths: []string{
			modutils.Dir("github.com/ironcore-dev/ironcore", "config", "apiserver", "apiservice", "bases"),
		},
		ErrorIfAPIServicePathIsMissing: true,
	}

	// Retrieve the binary directory to allow running tests from IDEs
	if dir := getEnvTestBinaryDir(); dir != "" {
		testEnv.BinaryAssetsDirectory = dir
	}

	var err error
	cfg, err = utilsenvtest.StartWithExtensions(testEnv, testEnvExt)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("starting the IronCore API server")
	apiSrv, err := apiserver.New(cfg, apiserver.Options{
		MainPath:     "github.com/ironcore-dev/ironcore/cmd/ironcore-apiserver",
		BuildOptions: []buildutils.BuildOption{buildutils.ModModeMod},
		ETCDServers:  []string{testEnv.ControlPlane.Etcd.URL.String()},
		Host:         testEnvExt.APIServiceInstallOptions.LocalServingHost,
		Port:         testEnvExt.APIServiceInstallOptions.LocalServingPort,
		CertDir:      testEnvExt.APIServiceInstallOptions.LocalServingCertDir,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(apiSrv.Start()).To(Succeed())
	DeferCleanup(apiSrv.Stop)

	err = utilsenvtest.WaitUntilAPIServicesReadyWithTimeout(apiServiceTimeout, testEnvExt, cfg, k8sClient, scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	Eventually(func() error {
		return utilsenvtest.StopWithExtensions(testEnv, testEnvExt)
	}, time.Minute, time.Second).Should(Succeed())
})

// getEnvTestBinaryDir locates the envtest binaries when the tests run directly (e.g., via an
// IDE) without the Makefile targets, which set KUBEBUILDER_ASSETS. To ensure the binaries are
// set up, run 'make setup-envtest' beforehand.
//
// A BinaryAssetsDirectory set here would take precedence over KUBEBUILDER_ASSETS, so this
// returns nothing when KUBEBUILDER_ASSETS is set. Otherwise it returns the newest directory in
// bin/k8s.
func getEnvTestBinaryDir() string {
	if os.Getenv("KUBEBUILDER_ASSETS") != "" {
		return ""
	}
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	dir := ""
	for _, entry := range entries {
		if entry.IsDir() {
			dir = filepath.Join(basePath, entry.Name())
		}
	}
	return dir
}
