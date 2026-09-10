package mutation

import (
	"context"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/resolver"
	_interface "jobqueue/interface"
)

type JobMutation struct {
	jobService _interface.JobService
	dataloader *_dataloader.GeneralDataloader
}

type enqueueArgs struct {
	Task string
}

func (q JobMutation) Enqueue(ctx context.Context, args enqueueArgs) (*resolver.JobResolver, error) {
	id, err := q.jobService.Enqueue(ctx, args.Task)
	if err != nil {
		return nil, err
	}
	return &resolver.JobResolver{
		JobID:      id,
		JobService: q.jobService,
		Dataloader: q.dataloader,
	}, nil
}

// NewJobMutation to create new instance
func NewJobMutation(jobService _interface.JobService, dataloader *_dataloader.GeneralDataloader) JobMutation {
	return JobMutation{
		jobService: jobService,
		dataloader: dataloader,
	}
}
