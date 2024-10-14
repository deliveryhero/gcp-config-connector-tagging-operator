package controller

import (
	"context"
	"fmt"
	"path/filepath"
	osruntime "runtime"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	storagev1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/storage/v1beta1"
	tagsv1alpha1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/tags/v1alpha1"
	tagmocks "github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/gcp/mocks"
	"github.com/deliveryhero/gcp-config-connector-tagging-operator/internal/util"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", osruntime.GOOS, osruntime.GOARCH)),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(tagsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(storagev1beta1.AddToScheme(scheme))

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())

	targetLabels := ".*"
	labelMatcher, err := util.LimitLabelsWithRegex(targetLabels)
	tagsManager := &tagmocks.TagsManager{}

	CreateTaggableResourceController(k8sManager, tagsManager, &StorageBucketMetadataProvider{}, labelMatcher)

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ ResourceMetadataProvider[storagev1beta1.StorageBucket] = &StorageBucketMetadataProvider{}

type StorageBucketMetadataProvider struct{}

func (in *StorageBucketMetadataProvider) GetResourceLocation(r *storagev1beta1.StorageBucket) string {
	return ptr.Deref(r.Spec.Location, "")
}

func (in *StorageBucketMetadataProvider) GetResourceID(_ string, r *storagev1beta1.StorageBucket) string {
	name := r.Name
	if r.Spec.ResourceID != nil {
		name = *r.Spec.ResourceID
	}
	return fmt.Sprintf("//storage.googleapis.com/projects/_/buckets/%s", name)
}
