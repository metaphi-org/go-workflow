package goworkflow_test

import (
	"context"
	"errors"
	"log"
	"math"
	"testing"
	"time"

	goworkflow "github.com/metaphi-org/go-workflow/go-workflow"
	"github.com/stretchr/testify/assert"
)

type Config struct{}

type Data struct {
	A        string
	B        string
	C        string
	Combined string
}

type customContext struct {
	context.Context
}

func getCustomContext() customContext {
	return customContext{context.TODO()}
}

func TestWorkflowCycleDetectionAndFailureHandling(t *testing.T) {
	ctx := getCustomContext()

	executeA := func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.A = "A"
		})
		return nil
	}
	executeB := func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.B = "B"
		})
		return nil
	}
	executeC := func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.C = "C"
		})
		return nil
	}
	executeD := func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B + d.C
		})
		return nil
	}

	wf := goworkflow.NewWorkflow[customContext, Config, Data](ctx)
	componentA := wf.AddComponent("A", executeA)
	componentB := wf.AddComponent("B", executeB)
	componentC := wf.AddComponent("C", executeC)
	componentD := wf.AddComponent("D", executeD)

	componentC.AddDependencies(componentA)
	componentD.AddDependencies(componentB, componentC)
	componentA.AddDependencies(componentD)

	_, st, err := wf.Execute(ctx, Config{}, &Data{})

	assert.Error(t, err)
	assert.Equal(t, st, goworkflow.ERROR)

	wf = goworkflow.NewWorkflow[customContext, Config, Data](ctx)

	componentA = wf.AddComponent("A", executeA)
	componentB = wf.AddComponent("B", executeB)
	componentC = wf.AddComponent("C", executeC)
	componentD = wf.AddComponent("D", executeD)

	componentD.AddDependencies(componentB, componentC, componentA)

	data, st, err := wf.Execute(ctx, Config{}, &Data{})

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
	assert.Equal(t, "ABC", data.Combined)

	wf = goworkflow.NewWorkflow[customContext, Config, Data](ctx)

	componentE := wf.AddComponent("E", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		return errors.New("this is not implemented")
	})

	componentF := wf.AddComponent("F", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = "F"
		})
		return nil
	})
	componentF.AddDependencies(componentE)
	_, st, err = wf.Execute(ctx, Config{}, &Data{})

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.ERROR)
	assert.Equal(t, goworkflow.ERROR, componentE.Status().Status)
	assert.NotEqual(t, "F", data.Combined)
	assert.Equal(t, goworkflow.ERROR, componentF.Status().Status)
}

func TestWorkflowTimes(t *testing.T) {
	ctx := getCustomContext()
	wf := goworkflow.NewWorkflow[customContext, Config, Data](ctx)
	wf.AddComponent("A", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(10 * time.Second)
		dt.Update(func(d *Data) {
			d.A = "A"
		})
		return nil
	})

	componentB := wf.AddComponent("B", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(2 * time.Second)
		dt.Update(func(d *Data) {
			d.B = "B"
		})
		return nil
	})

	componentC := wf.AddComponent("C", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(5 * time.Second)
		dt.Update(func(d *Data) {
			d.C = "C"
		})
		return nil
	})

	componentD := wf.AddComponent("D", func(ctx customContext, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B + d.C
		})
		return nil
	})

	componentC.AddDependencies(componentB)
	componentD.AddDependencies(componentC)

	startTimeSeconds := time.Now().Unix()
	data, st, err := wf.Execute(ctx, Config{}, &Data{})

	timeTakenSeconds := time.Now().Unix() - startTimeSeconds
	log.Println("Time taken:", timeTakenSeconds)
	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
	assert.Equal(t, "BC", data.Combined)
	assert.True(t, math.Abs(float64(timeTakenSeconds-10)) <= 0.01)
	assert.Equal(t, "A", data.A)
}

type Data2 struct {
	Run1     string
	Run2     string
	Combined string
}

func TestExample(t *testing.T) {
	wf1 := goworkflow.NewWorkflow[context.Context, Config, Data](context.TODO())
	cA1 := wf1.AddComponent("A1", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(5 * time.Second)
		dt.Update(func(d *Data) {
			d.A = "A1"
		})
		return nil
	})

	cB1 := wf1.AddComponent("B1", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.B = "B1"
		})
		return nil
	})

	cC1 := wf1.AddComponent("C1", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B
		})
		return nil
	})
	cC1.AddDependencies(cA1, cB1)

	wf2 := goworkflow.NewWorkflow[context.Context, Config, Data](context.TODO())
	cA2 := wf2.AddComponent("A2", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.A = "A2"
		})
		return nil
	})
	cB2 := wf2.AddComponent("B2", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data]) error {
		time.Sleep(2 * time.Second)
		dt.Update(func(d *Data) {
			d.B = "B2"
		})
		return nil
	})
	cC2 := wf2.AddComponent("C2", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data]) error {
		dt.Update(func(d *Data) {
			d.Combined = d.A + d.B
		})
		return nil
	})
	cC2.AddDependencies(cA2, cB2)

	wfCombined := goworkflow.NewWorkflow[context.Context, Config, Data2](context.TODO())
	c1 := wfCombined.AddComponent("WF1", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data2]) error {
		data := &Data{}
		_, _, err := wf1.Execute(ctx, Config{}, data)
		if err != nil {
			return err
		}
		dt.Update(func(d *Data2) {
			d.Run1 = data.Combined
		})
		return nil
	})
	c2 := wfCombined.AddComponent("WF2", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data2]) error {
		time.Sleep(4 * time.Second)
		data := &Data{}
		_, _, err := wf2.Execute(ctx, Config{}, data)
		if err != nil {
			return err
		}
		dt.Update(func(d *Data2) {
			d.Run2 = data.Combined
		})
		return nil
	})
	c3 := wfCombined.AddComponent("Combine", func(ctx context.Context, dt *goworkflow.DataTracker[Config, Data2]) error {
		dt.Update(func(d *Data2) {
			d.Combined = dt.GetData().Run1 + dt.GetData().Run2
		})
		return nil
	})
	c3.AddDependencies(c1, c2)

	startTime := time.Now().Unix()

	data, st, err := wfCombined.Execute(context.TODO(), Config{}, &Data2{})

	timeTaken := time.Now().Unix() - startTime

	assert.NoError(t, err)
	assert.Equal(t, st, goworkflow.DONE)
	assert.Equal(t, "A1B1A2B2", data.Combined)
	assert.True(t, math.Abs(float64(timeTaken-6)) <= 0.01)
}
