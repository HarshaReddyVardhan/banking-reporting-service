package report

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/banking/reporting-service/internal/domain"
)

type CSVGenerator struct{}

func (g *CSVGenerator) Generate(ctx context.Context, report *domain.Report, data interface{}, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Type assertion based on report type to write headers and rows
	switch report.Type {
	case domain.ReportTypeTransactionSummary:
		return g.writeTransactionSummary(writer, data)
	default:
		return fmt.Errorf("unsupported report type for CSV: %s", report.Type)
	}
}

func (g *CSVGenerator) writeTransactionSummary(w *csv.Writer, data interface{}) error {
	aggs, ok := data.([]domain.MetricAggregate)
	if !ok {
		return fmt.Errorf("invalid data type for transaction summary")
	}

	// Headers
	if err := w.Write([]string{"Bucket", "Count", "Total Amount", "Avg Latency (ms)"}); err != nil {
		return err
	}

	// Rows
	for _, agg := range aggs {
		row := []string{
			agg.Bucket.Format("2006-01-02 15:04:05"),
			strconv.FormatInt(agg.Count, 10),
			agg.TotalAmount.String(),
			fmt.Sprintf("%.2f", agg.AvgLatency),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
