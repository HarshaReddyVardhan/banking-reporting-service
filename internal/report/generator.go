package report

import (
	"context"
	"io"

	"github.com/banking/reporting-service/internal/domain"
)

type Generator interface {
	Generate(ctx context.Context, report *domain.Report, data interface{}, w io.Writer) error
}

type GeneratorFactory struct {
}

func NewGeneratorFactory() *GeneratorFactory {
	return &GeneratorFactory{}
}

func (f *GeneratorFactory) GetGenerator(format domain.ReportFormat) Generator {
	switch format {
	case domain.ReportFormatCSV:
		return &CSVGenerator{}
	// case domain.ReportFormatExcel:
	// 	return &ExcelGenerator{}
	default:
		return &CSVGenerator{}
	}
}
