package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archiver/v3"

	"cloud.google.com/go/storage"
	"github.com/pluralsh/plural/pkg/config"
	"github.com/pluralsh/plural/pkg/manifest"
	"github.com/pluralsh/plural/pkg/template"
	"github.com/pluralsh/plural/pkg/utils"
	"github.com/AlecAivazis/survey/v2"

	"k8s.io/api/core/v1"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

type GCPProvider struct {
	Clust         string `survey:"cluster"`
	Proj          string `survey:"project"`
	bucket        string
	region        string
	storageClient *storage.Client
	ctx           context.Context
}

var gcpSurvey = []*survey.Question{
	{
			Name:     "cluster",
			Prompt:   &survey.Input{Message: "Enter the name of your cluster"},
			Validate: utils.ValidateAlphaNumeric,
	},
	{
			Name: "project",
			Prompt: &survey.Input{Message: "Enter the name of its gcp project"},
			Validate: utils.ValidateAlphaNumeric,
	},
}

func mkGCP(conf config.Config) (*GCPProvider, error) {
	provider := &GCPProvider{}
	if err := survey.Ask(gcpSurvey, provider); err != nil {
		return nil, err
	}

	client, ctx, err := storageClient()
	if err != nil {
		return nil, err
	}
	provider.region = getRegion()
	provider.storageClient = client
	provider.ctx = ctx

	projectManifest := manifest.ProjectManifest{
		Cluster:  provider.Cluster(),
		Project:  provider.Project(),
		Provider: GCP,
		Region:   provider.Region(),
		Owner:    &manifest.Owner{Email: conf.Email, Endpoint: conf.Endpoint},
	}

	if err := projectManifest.Configure(); err != nil {
		return nil, err
	}

	provider.bucket = projectManifest.Bucket
	return provider, nil
}

func storageClient() (*storage.Client, context.Context, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	return client, ctx, err
}

func gcpFromManifest(man *manifest.Manifest) (*GCPProvider, error) {
	client, ctx, err := storageClient()
	if err != nil {
		return nil, err
	}

	region := man.Region
	if region == "" {
		region = "us-east1-b"
	}

	return &GCPProvider{man.Cluster, man.Project, man.Bucket, region, client, ctx}, nil
}

func (gcp *GCPProvider) KubeConfig() error {
	if utils.InKubernetes() {
		return nil
	}

	cmd := exec.Command(
		"gcloud", "container", "clusters", "get-credentials", gcp.Clust,
		"--region", getZone(gcp.region), "--project", gcp.Proj)
	return cmd.Run()
}

func (gcp *GCPProvider) CreateBackend(prefix string, ctx map[string]interface{}) (string, error) {
	if err := gcp.mkBucket(gcp.bucket); err != nil {
		return "", utils.ErrorWrap(err, "Failed to create terraform state bucket")
	}

	ctx["Project"] = gcp.Project()
	ctx["Location"] = gcp.Region()
	ctx["Bucket"] = gcp.Bucket()
	ctx["Prefix"] = prefix
	ctx["ClusterCreated"] = false
	ctx["__CLUSTER__"] = gcp.Cluster()
	if cluster, ok := ctx["cluster"]; ok {
		ctx["Cluster"] = cluster
		ctx["ClusterCreated"] = true
	} else {
		ctx["Cluster"] = fmt.Sprintf(`"%s"`, gcp.Cluster())
	}
	return template.RenderString(gcpBackendTemplate, ctx)
}

func (gcp *GCPProvider) mkBucket(name string) error {
	bkt := gcp.storageClient.Bucket(name)
	if _, err := bkt.Attrs(gcp.ctx); err != nil {
		return bkt.Create(gcp.ctx, gcp.Project(), nil)
	}
	return nil
}

func getRegion() string {
	cmd := exec.Command("gcloud", "config", "get-value", "compute/zone")
	res, err := cmd.CombinedOutput()
	if err != nil {
		return "us-east1-b"
	}

	return strings.Split(string(res), "\n")[1]
}

func getZone(region string) string {
	split := strings.Split(region, "-")
	return strings.Join(split[:2], "-")
}

func (gcp *GCPProvider) Install() (err error) {
	if exists, _ := utils.Which("gcloud"); exists {
		utils.Success("gcloud already installed!\n")
		return
	}

	goos := runtime.GOOS
	arch := runtime.GOARCH
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
		break
	case "arm64":
		arch = "arm"
	}

	url := fmt.Sprintf("https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-335.0.0-%s-%s.tar.gz", goos, arch)
	root, _ := utils.ProjectRoot()
	dest := filepath.Join(root, "gcloud-sdk.tar.gz")
	return utils.Install("gcloud", url, dest, func(dest string) (string, error) {
		gcloudPath := filepath.Join(filepath.Dir(dest), "gcloud-sdk")
		err := archiver.Unarchive(dest, gcloudPath)
		if err != nil {
			return "", err
		}

		installCommand := "install.sh"
		if goos == "windows" {
			installCommand = "install.bat"
		}

		cmd := exec.Command(filepath.Join(gcloudPath, installCommand), "--quiet")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return "", cmd.Run()
	})
}

func (gcp *GCPProvider) Name() string {
	return GCP
}

func (gcp *GCPProvider) Cluster() string {
	return gcp.Clust
}

func (gcp *GCPProvider) Project() string {
	return gcp.Proj
}

func (gcp *GCPProvider) Bucket() string {
	return gcp.bucket
}

func (gcp *GCPProvider) Region() string {
	return gcp.region
}

func (gcp *GCPProvider) Context() map[string]interface{} {
	return map[string]interface{}{}
}

func (gcp *GCPProvider) Decommision(node *v1.Node) error {
	ctx := context.Background()
	c, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return utils.ErrorWrap(err, "failed to initialize compute client")
	}
	defer c.Close()

	_, err = c.Delete(ctx, &computepb.DeleteInstanceRequest{
		Instance: node.Name,
		Project:  gcp.Project(),
		Zone:     gcp.Region(),
	})

	return utils.ErrorWrap(err, "failed to delete instance")
}
