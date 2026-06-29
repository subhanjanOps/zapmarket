package usecases_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zapmarket/zapmarket/services/import-service/internal/application/usecases"
)

type fakeCatalogClient struct{ created int }

func (f *fakeCatalogClient) BulkCreateProducts(_ context.Context, rows []usecases.ProductRow) (int, error) {
	f.created += len(rows)
	return len(rows), nil
}

type fakeJobRepo struct{ status string }

func (f *fakeJobRepo) UpdateStatus(_ context.Context, _, status string, _, _ int) error {
	f.status = status
	return nil
}

func TestProcessImport_BatchesRows(t *testing.T) {
	catalog := &fakeCatalogClient{}
	jobs := &fakeJobRepo{}
	uc := usecases.NewProcessImportUseCase(catalog, jobs, 100)

	csvData := "name,sku_code,price,category\n" +
		"Widget A,SKU001,999,Electronics\n" +
		"Widget B,SKU002,1499,Electronics\n"

	if err := uc.Execute(context.Background(), "job-1", strings.NewReader(csvData)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if catalog.created != 2 {
		t.Fatalf("expected 2 products created, got %d", catalog.created)
	}
	if jobs.status != "COMPLETE" {
		t.Fatalf("expected COMPLETE status, got %q", jobs.status)
	}
}

func TestProcessImport_BatchesBySize(t *testing.T) {
	catalog := &fakeCatalogClient{}
	jobs := &fakeJobRepo{}
	uc := usecases.NewProcessImportUseCase(catalog, jobs, 2) // batch size 2

	var rows []string
	for i := 0; i < 5; i++ {
		rows = append(rows, "Widget,SKU00X,100,Cat")
	}
	csvData := "name,sku_code,price,category\n" + strings.Join(rows, "\n")

	if err := uc.Execute(context.Background(), "job-2", strings.NewReader(csvData)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if catalog.created != 5 {
		t.Fatalf("expected 5 products created, got %d", catalog.created)
	}
}
