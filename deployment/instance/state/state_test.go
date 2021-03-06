package state_test

import (
	. "github.com/cloudfoundry/bosh-init/deployment/instance/state"

	biproperty "github.com/cloudfoundry/bosh-init/common/property"
	bias "github.com/cloudfoundry/bosh-init/deployment/applyspec"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("State", func() {
	Describe("ToApplySpec", func() {
		It("translates from instance model to apply spec model", func() {
			networkInterfaces := []NetworkRef{
				{
					Name: "fake-network-name",
					Interface: biproperty.Map{
						"ip":   "fake-ip",
						"type": "dynamic",
					},
				},
			}
			renderedJobs := []JobRef{
				{
					Name:    "fake-job-name",
					Version: "fake-job-fingerprint",
				},
			}
			compiledPackages := []PackageRef{
				{
					Name:    "vcloud_cpi",
					Version: "fake-fingerprint-cpi",
					Archive: BlobRef{
						SHA1:        "fake-sha1-cpi",
						BlobstoreID: "fake-package-blob-id-cpi",
					},
				},
				{
					Name:    "ruby",
					Version: "fake-fingerprint-ruby",
					Archive: BlobRef{
						SHA1:        "fake-sha1-ruby",
						BlobstoreID: "fake-package-blob-id-ruby",
					},
				},
			}
			renderedJobListBlob := BlobRef{
				BlobstoreID: "fake-rendered-job-list-archive-blob-id",
				SHA1:        "fake-rendered-job-list-archive-blob-sha1",
			}
			state := NewState(
				"fake-deployment-name",
				"fake-job-name",
				0,
				networkInterfaces,
				renderedJobs,
				compiledPackages,
				renderedJobListBlob,
				"fake-state-hash",
			)

			applySpec := state.ToApplySpec()

			Expect(applySpec).To(Equal(bias.ApplySpec{
				Deployment: "fake-deployment-name",
				Index:      0,
				Networks: map[string]biproperty.Map{
					"fake-network-name": biproperty.Map{
						"ip":   "fake-ip",
						"type": "dynamic",
					},
				},
				Job: bias.Job{
					Name: "fake-job-name",
					Templates: []bias.Blob{
						{
							Name:    "fake-job-name",
							Version: "fake-job-fingerprint",
						},
					},
				},
				Packages: map[string]bias.Blob{
					"vcloud_cpi": bias.Blob{
						Name:        "vcloud_cpi",
						Version:     "fake-fingerprint-cpi",
						SHA1:        "fake-sha1-cpi",
						BlobstoreID: "fake-package-blob-id-cpi",
					},
					"ruby": bias.Blob{
						Name:        "ruby",
						Version:     "fake-fingerprint-ruby",
						SHA1:        "fake-sha1-ruby",
						BlobstoreID: "fake-package-blob-id-ruby",
					},
				},
				RenderedTemplatesArchive: bias.RenderedTemplatesArchiveSpec{
					BlobstoreID: "fake-rendered-job-list-archive-blob-id",
					SHA1:        "fake-rendered-job-list-archive-blob-sha1",
				},
				ConfigurationHash: "fake-state-hash",
			}))
		})
	})

})
