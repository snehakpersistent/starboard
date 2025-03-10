package starboard

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"os"
	"testing"

	"github.com/aquasecurity/starboard/pkg/starboard"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kubeClient             client.Client
	apiextensionsClientset apiextensions.ApiextensionsV1beta1Interface
)

var (
	starboardCLILogLevel = "0"

	versionInfo = starboard.BuildInfo{
		Version: "dev",
		Commit:  "none",
		Date:    "unknown",
	}
	testNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "starboard-itest",
		},
	}

	conftestConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "starboard-conftest-config",
			Namespace: "starboard",
		},
		Data: map[string]string{
			"conftest.imageRef": "docker.io/openpolicyagent/conftest:v0.25.0",
			"conftest.policy.runs_as_root.rego": `
	package main
    
    deny[msg] {
      input.kind == "ReplicaSet"
      not input.spec.template.spec.securityContext.runAsNonRoot

      msg := "Containers must not run as root"
    }

    deny[msg] {
      input.kind == "Pod"
      not input.spec.securityContext.runAsNonRoot

      msg := "Containers must not run as root"
    }

    deny[msg] {
      input.kind == "CronJob"
      not input.spec.jobTemplate.spec.template.spec.securityContext.runAsNonRoot

      msg := "Containers must not run as root"
    }`,
		},
	}
)

var (
	customResourceDefinitions apiextensions.CustomResourceDefinitionInterface
)

// TestStarboardCLI is a spec that describes the behavior of Starboard CLI.
func TestStarboardCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Starboard CLI")
}

var _ = BeforeSuite(func() {
	var err error

	klog.InitFlags(nil)

	if logLevel, ok := os.LookupEnv("STARBOARD_CLI_LOG_LEVEL"); ok {
		starboardCLILogLevel = logLevel
	}

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	Expect(err).ToNot(HaveOccurred())

	kubeClient, err = client.New(config, client.Options{
		Scheme: starboard.NewScheme(),
	})
	Expect(err).ToNot(HaveOccurred())

	apiextensionsClientset, err = apiextensions.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())

	customResourceDefinitions = apiextensionsClientset.CustomResourceDefinitions()

	err = kubeClient.Create(context.Background(), testNamespace)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := kubeClient.Delete(context.Background(), testNamespace)
	Expect(err).ToNot(HaveOccurred())
	klog.Flush()
})
