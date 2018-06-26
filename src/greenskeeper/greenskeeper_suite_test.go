package greenskeeper

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	sleepyBin  string
	sleepyBin2 string
)

func TestGreenskeeper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Greenskeeper Suite")
}

var _ = BeforeSuite(func() {
	var err error
	sleepyBin, err = gexec.Build("greenskeeper/integration/assets/sleepy")
	Expect(err).ToNot(HaveOccurred())
	sleepyBin2, err = gexec.Build("greenskeeper/integration/assets/sleepy")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
