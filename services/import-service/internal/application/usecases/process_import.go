package usecases

import (
	"context"
	"encoding/csv"
	"io"
)

type ProductRow struct {
	Name     string
	SKUCode  string
	Price    string
	Category string
}

type catalogClient interface {
	BulkCreateProducts(ctx context.Context, rows []ProductRow) (created int, err error)
}

type jobUpdater interface {
	UpdateStatus(ctx context.Context, jobID, status string, processed, failed int) error
}

type ProcessImportUseCase struct {
	catalog   catalogClient
	jobs      jobUpdater
	batchSize int
}

func NewProcessImportUseCase(catalog catalogClient, jobs jobUpdater, batchSize int) *ProcessImportUseCase {
	return &ProcessImportUseCase{catalog: catalog, jobs: jobs, batchSize: batchSize}
}

func (uc *ProcessImportUseCase) Execute(ctx context.Context, jobID string, r io.Reader) error {
	reader := csv.NewReader(r)
	reader.Read() //nolint:errcheck // skip header row

	var batch []ProductRow
	total, failed := 0, 0

	flush := func() {
		if len(batch) == 0 {
			return
		}
		created, _ := uc.catalog.BulkCreateProducts(ctx, batch)
		total += created
		failed += len(batch) - created
		batch = batch[:0]
		uc.jobs.UpdateStatus(ctx, jobID, "IN_PROGRESS", total, failed) //nolint:errcheck
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 4 {
			failed++
			continue
		}
		batch = append(batch, ProductRow{
			Name:     record[0],
			SKUCode:  record[1],
			Price:    record[2],
			Category: record[3],
		})
		if len(batch) >= uc.batchSize {
			flush()
		}
	}
	flush()

	status := "COMPLETE"
	if failed > 0 && total == 0 {
		status = "FAILED"
	}
	return uc.jobs.UpdateStatus(ctx, jobID, status, total, failed)
}
