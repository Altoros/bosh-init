package tarball_test

import (
	"errors"
	"io/ioutil"
	"os"

	fakebicrypto "github.com/cloudfoundry/bosh-init/crypto/fakes"
	fakebihttpclient "github.com/cloudfoundry/bosh-init/deployment/httpclient/fakes"
	. "github.com/cloudfoundry/bosh-init/installation/tarball"
	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider", func() {
	var (
		provider       Provider
		cache          Cache
		fs             *fakesys.FakeFileSystem
		httpClient     *fakebihttpclient.FakeHTTPClient
		sha1Calculator *fakebicrypto.FakeSha1Calculator
		source         *fakeSource
		fakeStage      *fakebiui.FakeStage
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		cache = NewCache("/fake-base-path", fs, logger)
		sha1Calculator = fakebicrypto.NewFakeSha1Calculator()
		httpClient = fakebihttpclient.NewFakeHTTPClient()
		provider = NewProvider(cache, fs, httpClient, sha1Calculator, 3, 0, logger)
		fakeStage = fakebiui.NewFakeStage()
	})

	Describe("Get", func() {
		Context("when URL starts with file://", func() {
			BeforeEach(func() {
				source = newFakeSource("file://fake-file", "fake-sha1", "fake-description")
				fs.WriteFileString("expanded-file-path", "")
				fs.ExpandPathExpanded = "expanded-file-path"
			})

			It("returns expanded path to file", func() {
				path, err := provider.Get(source, fakeStage)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("expanded-file-path"))
			})
		})

		Context("when URL starts with http(s)://", func() {
			BeforeEach(func() {
				source = newFakeSource("http://fake-url", "fake-sha1", "fake-description")
			})

			Context("when tarball is present in cache", func() {
				BeforeEach(func() {
					fs.WriteFileString("fake-source-path", "")
					cache.Save("fake-source-path", source)
				})

				It("returns cached tarball path", func() {
					path, err := provider.Get(source, fakeStage)
					Expect(err).ToNot(HaveOccurred())
					Expect(path).To(Equal("/fake-base-path/9db1fb7c47637e8709e944a232e1aa98ce6fec26-fake-sha1"))
				})

				It("skips downloading stage", func() {
					_, err := provider.Get(source, fakeStage)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeStage.PerformCalls[0].Name).To(Equal("Downloading fake-description"))
					Expect(fakeStage.PerformCalls[0].SkipError.Error()).To(Equal("Found in local cache: Already downloaded"))
				})
			})

			Context("when tarball is not present in cache", func() {
				var (
					tempDownloadFilePath string
				)

				BeforeEach(func() {
					tempDownloadFile, err := ioutil.TempFile("", "temp-download-file")
					Expect(err).ToNot(HaveOccurred())
					fs.ReturnTempFile = tempDownloadFile
					tempDownloadFilePath = tempDownloadFile.Name()
					sha1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
						tempDownloadFilePath: {Sha1: "fake-sha1"},
					})
				})

				AfterEach(func() {
					os.RemoveAll(tempDownloadFilePath)
				})

				Context("when downloading succeds", func() {
					BeforeEach(func() {
						httpClient.SetGetBehavior("fake-body", 200, nil)
						httpClient.SetGetBehavior("fake-body", 200, nil)
						httpClient.SetGetBehavior("fake-body", 200, nil)
					})

					It("downloads tarball from given URL and returns saved cache tarball path", func() {
						path, err := provider.Get(source, fakeStage)
						Expect(err).ToNot(HaveOccurred())
						Expect(path).To(Equal("/fake-base-path/9db1fb7c47637e8709e944a232e1aa98ce6fec26-fake-sha1"))

						Expect(httpClient.GetInputs).To(HaveLen(1))
						Expect(httpClient.GetInputs[0].Endpoint).To(Equal("http://fake-url"))
					})

					It("logs downloading stage", func() {
						_, err := provider.Get(source, fakeStage)
						Expect(err).ToNot(HaveOccurred())

						Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
							{Name: "Downloading fake-description"},
						}))
					})

					Context("when sha1 does not match", func() {
						BeforeEach(func() {
							sha1Calculator.SetCalculateBehavior(map[string]fakebicrypto.CalculateInput{
								tempDownloadFilePath: {Sha1: "fake-sha2"},
							})
						})

						It("returns an error", func() {
							_, err := provider.Get(source, fakeStage)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("'fake-sha2' does not match expected SHA1 'fake-sha1'"))
						})

						It("retries downloading up to 3 times", func() {
							_, err := provider.Get(source, fakeStage)
							Expect(err).To(HaveOccurred())

							Expect(httpClient.GetInputs).To(HaveLen(3))
						})

						It("removes the downloaded file", func() {
							_, err := provider.Get(source, fakeStage)
							Expect(err).To(HaveOccurred())
							Expect(fs.FileExists(tempDownloadFilePath)).To(BeFalse())
						})
					})

					Context("when saving to cache fails", func() {
						BeforeEach(func() {
							// Creating cache base directory fails
							fs.MkdirAllError = errors.New("fake-mkdir-error")
						})

						It("returns an error", func() {
							_, err := provider.Get(source, fakeStage)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("fake-mkdir-error"))
						})

						It("removes the downloaded file", func() {
							_, err := provider.Get(source, fakeStage)
							Expect(err).To(HaveOccurred())
							Expect(fs.FileExists(tempDownloadFilePath)).To(BeFalse())
						})
					})
				})

				Context("when downloading fails", func() {
					BeforeEach(func() {
						httpClient.SetGetBehavior("", 500, errors.New("fake-download-error-1"))
						httpClient.SetGetBehavior("", 500, errors.New("fake-download-error-2"))
						httpClient.SetGetBehavior("", 500, errors.New("fake-download-error-3"))
					})

					It("retries downloading up to 3 times", func() {
						_, err := provider.Get(source, fakeStage)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-download-error-3"))

						Expect(httpClient.GetInputs).To(HaveLen(3))
					})

					It("removes the downloaded file", func() {
						_, err := provider.Get(source, fakeStage)
						Expect(err).To(HaveOccurred())
						Expect(fs.FileExists(tempDownloadFilePath)).To(BeFalse())
					})
				})
			})
		})

		Context("when URL does not start with either file:// or http(s)://", func() {
			BeforeEach(func() {
				source = newFakeSource("invalid-url", "fake-sha1", "fake-description")
			})

			It("returns an error", func() {
				_, err := provider.Get(source, fakeStage)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Invalid source URL: 'invalid-url'"))
			})
		})
	})
})

type fakeSource struct {
	url         string
	sha1        string
	description string
}

func newFakeSource(url, sha1, description string) *fakeSource {
	return &fakeSource{url, sha1, description}
}

func (s *fakeSource) GetURL() string      { return s.url }
func (s *fakeSource) GetSHA1() string     { return s.sha1 }
func (s *fakeSource) Description() string { return s.description }
