package common

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func generateResourceOffer() []*mesos.Resource {
	val := []*mesos.Resource{
		util.NewScalarResource("cpus", 3),
		util.NewScalarResource("disk", 73590),
		util.NewScalarResource("mem", 1985),
		util.NewRangesResource("ports", []*mesos.Value_Range{util.NewValueRange(31000, 32000)}),
	}
	return val
}
func TestGoodCPUAsk(t *testing.T) {
	assert := assert.New(t)
	offer2 := generateResourceOffer()
	offer := generateResourceOffer()
	askFun := AskForCPU(1)
	newOffer, resourceAsk, success := askFun(offer)
	assert.Equal(true, success)
	assert.Equal(util.NewScalarResource("cpus", 1), resourceAsk)
	assert.Equal(offer, offer2)
	cpuResources := util.FilterResources(newOffer, func(res *mesos.Resource) bool {
		return res.GetName() == "cpus"
	})
	assert.Equal(2.0, cpuResources[0].Scalar.GetValue())
}

func TestTotalCPUAsk(t *testing.T) {
	assert := assert.New(t)
	offer2 := generateResourceOffer()
	offer := generateResourceOffer()
	askFun := AskForCPU(3)
	newOffer, resourceAsk, success := askFun(offer)
	assert.Equal(true, success)
	assert.Equal(util.NewScalarResource("cpus", 3), resourceAsk)
	assert.Equal(offer, offer2)
	cpuResources := util.FilterResources(newOffer, func(res *mesos.Resource) bool {
		return res.GetName() == "cpus"
	})
	assert.Equal(0.0, cpuResources[0].Scalar.GetValue())
}

func TestBadCPUAsk(t *testing.T) {
	assert := assert.New(t)
	offer2 := generateResourceOffer()
	offer := generateResourceOffer()
	askFun := AskForCPU(3.1)
	newOffer, _, success := askFun(offer)
	assert.Equal(false, success)
	assert.Equal(offer, offer2)
	cpuResources := util.FilterResources(newOffer, func(res *mesos.Resource) bool {
		return res.GetName() == "cpus"
	})
	assert.Equal(3.0, cpuResources[0].Scalar.GetValue())
}

func TestGoodMemoryAsk(t *testing.T) {
	assert := assert.New(t)
	offer2 := generateResourceOffer()
	offer := generateResourceOffer()
	askFun := AskForMemory(100)
	newOffer, resourceAsk, success := askFun(offer)
	assert.Equal(true, success)
	assert.Equal(util.NewScalarResource("mem", 100), resourceAsk)
	assert.Equal(offer, offer2)
	cpuResources := util.FilterResources(newOffer, func(res *mesos.Resource) bool {
		return res.GetName() == "mem"
	})
	assert.Equal(1885.0, cpuResources[0].Scalar.GetValue())
}

func TestArrayToRanges(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(arrayToRanges([]int64{}), []*mesos.Value_Range{})
	assert.Equal(arrayToRanges([]int64{1, 2, 3, 4}), []*mesos.Value_Range{util.NewValueRange(1, 4)})
	assert.Equal(arrayToRanges([]int64{1, 2, 3, 4, 6, 7, 8}), []*mesos.Value_Range{util.NewValueRange(1, 4), util.NewValueRange(6, 8)})
	assert.Equal(arrayToRanges([]int64{2, 3, 4, 6, 7, 8}), []*mesos.Value_Range{util.NewValueRange(2, 4), util.NewValueRange(6, 8)})
	assert.Equal(arrayToRanges([]int64{1, 3, 5}), []*mesos.Value_Range{util.NewValueRange(1, 1), util.NewValueRange(3, 3), util.NewValueRange(5, 5)})
}

func TestGoodPortAsk(t *testing.T) {
	assert := assert.New(t)
	offer := generateResourceOffer()
	askFun := AskForPorts(100)
	_, resourceAsk, success := askFun(offer)
	assert.Equal(true, success)
	assert.Equal(util.NewRangesResource("ports", []*mesos.Value_Range{util.NewValueRange(31000, 31099)}), resourceAsk)
}

func TestBadPortAsk(t *testing.T) {
	assert := assert.New(t)
	offer := []*mesos.Resource{util.NewRangesResource("ports", []*mesos.Value_Range{util.NewValueRange(31000, 31000)})}
	_, _, success := AskForPorts(100)(offer)

	assert.Equal(false, success)
}

func TestTotalPortAsk(t *testing.T) {
	assert := assert.New(t)
	askfun := AskForPorts(1)
	offer := []*mesos.Resource{util.NewRangesResource("ports", []*mesos.Value_Range{util.NewValueRange(31000, 31000)})}
	newOffer, _, success := askfun(offer)
	newOffer[0].GetRanges().GetRange()
	assert.Equal(0, len(newOffer[0].GetRanges().GetRange()))
	assert.Equal(true, success)
}
