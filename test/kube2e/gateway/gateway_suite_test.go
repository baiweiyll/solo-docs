package gateway_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solo-io/gloo/test/helpers"

	"github.com/avast/retry-go"
	"github.com/solo-io/gloo/test/kube2e"
	"github.com/solo-io/go-utils/testutils/clusterlock"

	"github.com/solo-io/go-utils/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils/helper"
	skhelpers "github.com/solo-io/solo-kit/test/helpers"
)

func TestGateway(t *testing.T) {
	if testutils.AreTestsDisabled() {
		return
	}
	skhelpers.RegisterCommonFailHandlers()
	skhelpers.SetupLog()
	RunSpecs(t, "Gateway Suite")
}

var testHelper *helper.SoloTestHelper
var locker *clusterlock.TestClusterLocker

var _ = BeforeSuite(func() {
	cwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	testHelper, err = helper.NewSoloTestHelper(func(defaults helper.TestConfig) helper.TestConfig {
		defaults.RootDir = filepath.Join(cwd, "../../..")
		defaults.HelmChartName = "gloo"
		return defaults
	})
	Expect(err).NotTo(HaveOccurred())

	RegisterFailHandler(helpers.KubeDumpOnFail(GinkgoWriter, "knative-serving", testHelper.InstallNamespace))
	testHelper.Verbose = true

	locker, err = clusterlock.NewTestClusterLocker(kube2e.MustKubeClient(), clusterlock.Options{})
	Expect(err).NotTo(HaveOccurred())
	Expect(locker.AcquireLock(retry.Attempts(40))).NotTo(HaveOccurred())

	err = testHelper.InstallGloo(helper.GATEWAY, 5*time.Minute)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	if locker != nil {
		defer locker.ReleaseLock()
	}

	if testHelper != nil {
		err := testHelper.UninstallGlooAll()
		Expect(err).NotTo(HaveOccurred())

		// TODO go-utils should expose `glooctl uninstall --delete-namespace`
		_ = testutils.Kubectl("delete", "namespace", testHelper.InstallNamespace)

		Eventually(func() error {
			return testutils.Kubectl("get", "namespace", testHelper.InstallNamespace)
		}, "60s", "1s").Should(HaveOccurred())
	}
})
