package goworkflow_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	goworkflow "github.com/metaphi-org/go-workflow/go-workflow"
	"github.com/metaphi-org/go-workflow/go-workflow/limiter"
	"github.com/stretchr/testify/assert"
)

/*
Examples for go-workflow package
*/

type PageAnalysisConfig struct {
	PdfPage []byte
}
type PageAnalysisDataStore struct {
	VisualInformation []string
	ExtractedText     []string

	Parameter1 string
	Parameter2 string
	Parameter3 string
}

func getPageAnalysis(ctx context.Context, page []byte) (*PageAnalysisDataStore, goworkflow.Status, error) {
	pageAnalysisWorkflow := goworkflow.NewWorkflow[context.Context, PageAnalysisConfig, PageAnalysisDataStore](ctx)

	visualInfomationComponent := pageAnalysisWorkflow.AddComponent(
		goworkflow.MakeComponent(
			"VisualInfomationExtraction",
			nil,
			func(ctx context.Context, input any, dt *goworkflow.DataTracker[PageAnalysisConfig, PageAnalysisDataStore]) error {
				// prepare visual information
				time.Sleep(1 * time.Second)
				dt.Update(func(data *PageAnalysisDataStore) {
					data.VisualInformation = []string{"visual1", "visual2", "visual3"}
				})
				return nil
			},
		),
	)

	textExtractorComponent := pageAnalysisWorkflow.AddComponent(
		goworkflow.MakeComponent(
			"TextExtractor",
			nil,
			func(ctx context.Context, input any, dt *goworkflow.DataTracker[PageAnalysisConfig, PageAnalysisDataStore]) error {
				// prepare extracted text
				time.Sleep(1 * time.Second)
				dt.Update(func(data *PageAnalysisDataStore) {
					data.ExtractedText = []string{"text1", "text2", "text3"}
				})
				return nil
			},
		),
	)

	parameter1Component := pageAnalysisWorkflow.AddComponent(
		goworkflow.MakeComponent(
			"Parameter1",
			nil,
			func(ctx context.Context, input any, dt *goworkflow.DataTracker[PageAnalysisConfig, PageAnalysisDataStore]) error {
				// prepare parameter1
				time.Sleep(1 * time.Second)
				dt.Update(func(data *PageAnalysisDataStore) {
					data.Parameter1 = data.VisualInformation[0] + data.ExtractedText[0]
				})
				return nil
			},
		),
	)
	parameter1Component.AddDependencies(visualInfomationComponent, textExtractorComponent)

	parameter2Component := pageAnalysisWorkflow.AddComponent(
		goworkflow.MakeComponent(
			"Parameter2",
			nil,
			func(ctx context.Context, input any, dt *goworkflow.DataTracker[PageAnalysisConfig, PageAnalysisDataStore]) error {
				// prepare parameter2
				time.Sleep(4 * time.Second)
				dt.Update(func(data *PageAnalysisDataStore) {
					data.Parameter2 = data.VisualInformation[1] + data.ExtractedText[1]
				})
				return nil
			},
		),
	)
	parameter2Component.AddDependencies(visualInfomationComponent, textExtractorComponent)

	parameter3Component := pageAnalysisWorkflow.AddComponent(
		goworkflow.MakeComponent(
			"Parameter3",
			nil,
			func(ctx context.Context, input any, dt *goworkflow.DataTracker[PageAnalysisConfig, PageAnalysisDataStore]) error {
				// prepare parameter3
				time.Sleep(3 * time.Second)
				dt.Update(func(data *PageAnalysisDataStore) {
					data.Parameter3 = data.VisualInformation[2] + data.ExtractedText[2]
				})
				return nil
			},
		),
	)
	parameter3Component.AddDependencies(visualInfomationComponent, textExtractorComponent)

	config := PageAnalysisConfig{PdfPage: page}
	data := PageAnalysisDataStore{}

	dt, status, err := pageAnalysisWorkflow.Execute(ctx, config, &data)
	return dt, status, err
}

type DocumentAnalysisConfig struct {
	PdfDocument [][]byte
}

type DocumentAnalysisDataStore struct {
	PageAnalysisData []*PageAnalysisDataStore
}

type DocumentAnalysisInput struct {
	Index int
}

func getDocumentAnalysis(ctx context.Context, pages [][]byte, maxConcurrency int) (*DocumentAnalysisDataStore, goworkflow.Status, error) {
	noOfPages := len(pages)
	pageConcurrencyLimiter := limiter.NewConcurrencyLimiter(maxConcurrency)
	documentAnalysisWorkflow := goworkflow.NewWorkflow[context.Context, DocumentAnalysisConfig, DocumentAnalysisDataStore](ctx)

	for i := 0; i < noOfPages; i++ {
		documentAnalysisWorkflow.AddComponent(
			goworkflow.MakeComponent(
				"PageAnalysis",
				DocumentAnalysisInput{Index: i},
				func(ctx context.Context, input DocumentAnalysisInput, dt *goworkflow.DataTracker[DocumentAnalysisConfig, DocumentAnalysisDataStore]) error {
					res, status, err := getPageAnalysis(ctx, dt.Config.PdfDocument[input.Index])
					if err != nil {
						return err
					}
					if status != goworkflow.DONE {
						return errors.New("page analysis failed")
					}
					dt.Update(func(data *DocumentAnalysisDataStore) {
						data.PageAnalysisData[input.Index] = res
					})
					return nil
				},
			),
			&goworkflow.AddComponentConfig{
				ConcurrencyLimiter: pageConcurrencyLimiter,
			},
		)
	}

	config := DocumentAnalysisConfig{PdfDocument: pages}
	data := DocumentAnalysisDataStore{PageAnalysisData: make([]*PageAnalysisDataStore, noOfPages)}

	d, st, err := documentAnalysisWorkflow.Execute(ctx, config, &data)
	return d, st, err
}

func TestExampleDocumentAI(t *testing.T) {
	ctx := context.Background()
	startTime := time.Now()
	dt, status, err := getPageAnalysis(ctx, []byte("pdf page content"))
	elapsedTime := time.Since(startTime)

	// max time per page -> max(1 sec (visual information) + 1 sec (text extraction)) + max(1 sec (parameter1) + 4 sec (parameter2) + 3 sec (parameter3)) = 5 sec
	assert.True(t, math.Abs(float64(elapsedTime.Milliseconds()-5*1000)) < 100, "elapsed time should be around 5 seconds")
	assert.NoError(t, err)
	assert.Equal(t, goworkflow.DONE, status)
	assert.Equal(t, "visual1text1", dt.Parameter1)
	assert.Equal(t, "visual2text2", dt.Parameter2)
	assert.Equal(t, "visual3text3", dt.Parameter3)

	allPages := [][]byte{}
	for i := 0; i < 100; i++ {
		allPages = append(allPages, []byte("pdf page content"))
	}

	startTime = time.Now()
	maxConcurrency := 50
	dt2, status, err := getDocumentAnalysis(ctx, allPages, maxConcurrency)
	elapsedTime = time.Since(startTime)

	maxTime := (len(allPages) / maxConcurrency) * 5
	assert.True(t, math.Abs(float64(elapsedTime.Milliseconds()-int64(maxTime*1000))) < 100, "elapsed time should be around %d seconds", maxTime)
	assert.NoError(t, err)
	assert.Equal(t, goworkflow.DONE, status)
	assert.Len(t, dt2.PageAnalysisData, 100)

}
