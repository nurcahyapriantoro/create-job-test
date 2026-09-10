package _dataloader

import (
	"context"

	"github.com/graph-gophers/dataloader/v6"
)

func (s GeneralDataloader) JobBatchFunc(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	for i, key := range keys {
		_ = key
		results[i] = &dataloader.Result{}
	}
	return results
}
