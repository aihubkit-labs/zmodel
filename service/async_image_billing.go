package service

import (
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

type AsyncImageBillingSettler struct {
	TaskID           string
	PreConsumedQuota int
	LeaseOwner       string
	Manifest         string
	SourceKind       string
	Objects          []model.StorageObject
	ArchiveErrorCode string
	ArchiveError     string
}

func (settler *AsyncImageBillingSettler) Settle(actualQuota int) error {
	if settler.Manifest != "" && len(settler.Objects) > 0 {
		return model.CompleteAsyncImageGeneration(settler.TaskID, settler.LeaseOwner, actualQuota, settler.Manifest, settler.SourceKind, settler.Objects)
	}
	if settler.ArchiveErrorCode != "" {
		return model.CompleteAsyncImageGenerationWithArchiveFailure(settler.TaskID, settler.LeaseOwner, actualQuota, settler.SourceKind, settler.ArchiveErrorCode, settler.ArchiveError)
	}
	return model.SettleAsyncImageBilling(settler.TaskID, actualQuota)
}

func (settler *AsyncImageBillingSettler) Refund(_ *gin.Context) {
	gopool.Go(func() {
		_ = model.RefundAsyncImageBilling(settler.TaskID, "image_generation_failed", "image generation failed")
	})
}

func (settler *AsyncImageBillingSettler) NeedsRefund() bool {
	task, err := model.GetAsyncImageTaskByTaskID(settler.TaskID)
	return err == nil && task.BillingStatus == model.AsyncImageBillingReserved
}

func (settler *AsyncImageBillingSettler) GetPreConsumedQuota() int {
	return settler.PreConsumedQuota
}

func (settler *AsyncImageBillingSettler) Reserve(targetQuota int) error {
	return model.ReserveAsyncImageBilling(settler.TaskID, targetQuota)
}
