package state

import (
	biblobstore "github.com/cloudfoundry/bosh-init/blobstore"
	biagentclient "github.com/cloudfoundry/bosh-init/deployment/agentclient"
	bideplrel "github.com/cloudfoundry/bosh-init/deployment/release"
	bistatejob "github.com/cloudfoundry/bosh-init/state/job"
	bistatepkg "github.com/cloudfoundry/bosh-init/state/pkg"
	bitemplate "github.com/cloudfoundry/bosh-init/templatescompiler"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type BuilderFactory interface {
	NewBuilder(biblobstore.Blobstore, biagentclient.AgentClient) Builder
}

type builderFactory struct {
	packageRepo               bistatepkg.CompiledPackageRepo
	releaseJobResolver        bideplrel.JobResolver
	jobRenderer               bitemplate.JobListRenderer
	renderedJobListCompressor bitemplate.RenderedJobListCompressor
	logger                    boshlog.Logger
}

func NewBuilderFactory(
	packageRepo bistatepkg.CompiledPackageRepo,
	releaseJobResolver bideplrel.JobResolver,
	jobRenderer bitemplate.JobListRenderer,
	renderedJobListCompressor bitemplate.RenderedJobListCompressor,
	logger boshlog.Logger,
) BuilderFactory {
	return &builderFactory{
		packageRepo:               packageRepo,
		releaseJobResolver:        releaseJobResolver,
		jobRenderer:               jobRenderer,
		renderedJobListCompressor: renderedJobListCompressor,
		logger: logger,
	}
}

func (f *builderFactory) NewBuilder(blobstore biblobstore.Blobstore, agentClient biagentclient.AgentClient) Builder {
	packageCompiler := NewRemotePackageCompiler(blobstore, agentClient, f.packageRepo)
	jobDependencyCompiler := bistatejob.NewDependencyCompiler(packageCompiler, f.logger)

	return NewBuilder(
		f.releaseJobResolver,
		jobDependencyCompiler,
		f.jobRenderer,
		f.renderedJobListCompressor,
		blobstore,
		f.logger,
	)
}
