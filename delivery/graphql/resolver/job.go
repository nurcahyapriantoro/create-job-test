package resolver

import (
	"context"

	_dataloader "jobqueue/delivery/graphql/dataloader"
	_interface "jobqueue/interface"
)

type JobResolver struct {
	JobID      string
	JobService _interface.JobService
	Dataloader *_dataloader.GeneralDataloader
}

type JobStatusResolver struct {
	JobService _interface.JobService
	Dataloader *_dataloader.GeneralDataloader
}

func (q JobResolver) ID() string {
	return q.JobID
}

func (q JobResolver) Task() string {
	job, err := q.JobService.FindByID(context.Background(), q.JobID)
	if err != nil {
		return ""
	}
	return job.Task
}

func (q JobResolver) Status() string {
	job, err := q.JobService.FindByID(context.Background(), q.JobID)
	if err != nil {
		return ""
	}
	return job.Status
}

func (q JobResolver) Attempts() int32 {
	job, err := q.JobService.FindByID(context.Background(), q.JobID)
	if err != nil {
		return 0
	}
	return job.Attempts
}

func (t JobStatusResolver) Pending() int32 {
	status, _ := t.JobService.GetJobStatus(context.Background())
	return status.Pending
}

func (t JobStatusResolver) Running() int32 {
	status, _ := t.JobService.GetJobStatus(context.Background())
	return status.Running
}

func (t JobStatusResolver) Failed() int32 {
	status, _ := t.JobService.GetJobStatus(context.Background())
	return status.Failed
}

func (t JobStatusResolver) Completed() int32 {
	status, _ := t.JobService.GetJobStatus(context.Background())
	return status.Completed
}
