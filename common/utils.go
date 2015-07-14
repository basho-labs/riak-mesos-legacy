package common

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"sort"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}
// The new value for the resource, the ask, and whether or not the ask was accomodated - the ask may be nil if it wasn't accomodated
type ResourceAsker (func([]*mesos.Resource) (remaining []*mesos.Resource, ask *mesos.Resource, success bool))
type CombinedResourceAsker (func([]*mesos.Resource) (remaining []*mesos.Resource, ask []*mesos.Resource, success bool))

func AskForScalar(resourceName string, askSize float64) ResourceAsker {
	return func(resources []*mesos.Resource) ([]*mesos.Resource, *mesos.Resource, bool) {
		newResources := make([]*mesos.Resource, len(resources))
		copy(newResources, resources)
		for idx, resource := range resources {
			if resource.GetName() == resourceName && askSize <= resource.GetScalar().GetValue() {
				newResources[idx] = util.NewScalarResource(resourceName, resource.GetScalar().GetValue()-askSize)
				ask := util.NewScalarResource(resourceName, askSize)
				return newResources, ask, true
			}
		}
		return newResources, nil, false
	}
}
func AskForCPU(cpuAsk float64) ResourceAsker {
	return AskForScalar("cpus", cpuAsk)
}

func AskForMemory(memory float64) ResourceAsker {
	return AskForScalar("mem", memory)

}

func AskForDisk(disk float64) ResourceAsker {
	return AskForScalar("disk", disk)
}

type intarray []int64

func (a intarray) Len() int           { return len(a) }
func (a intarray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a intarray) Less(i, j int) bool { return a[i] < a[j] }

// We assume the input is sorted
func ArrayToRanges(ports []int64) []*mesos.Value_Range {
	sort.Sort(intarray(ports))
	if len(ports) == 0 {
		return []*mesos.Value_Range{}
	}
	fakeret := [][]int64{[]int64{ports[0], ports[0]}}
	for _, val := range ports {
		if val > fakeret[len(fakeret)-1][1]+1 {
			fakeret = append(fakeret, []int64{val, val})
		} else {
			fakeret[len(fakeret)-1][1] = val
		}
	}
	ret := make([]*mesos.Value_Range, len(fakeret))
	for idx := range fakeret {
		ret[idx] = util.NewValueRange(uint64(fakeret[idx][0]), uint64(fakeret[idx][1]))
	}
	return ret
}
func RangesToArray(ranges []*mesos.Value_Range) []int64 {
	array := make([]int64, 0)
	for _, mesosRange := range ranges {
		temp := make([]int64, mesosRange.GetEnd()-mesosRange.GetBegin()+1)
		idx := 0
		for i := mesosRange.GetBegin(); i <= mesosRange.GetEnd(); i++ {
			temp[idx] = int64(i)
			idx++
		}
		array = append(array, temp...)
	}
	return array
}

func AskForPorts(portCount int) ResourceAsker {
	ret := func(resources []*mesos.Resource) ([]*mesos.Resource, *mesos.Resource, bool) {
		newResources := make([]*mesos.Resource, len(resources))
		copy(newResources, resources)
		for idx, resource := range resources {
			if resource.GetName() == "ports" {
				ports := RangesToArray(resource.GetRanges().GetRange())
				// Now we have to see if there are N ports
				if len(ports) >= portCount {
					takingPorts := ports[:portCount]
					leavingPorts := ports[portCount:]
					newResources[idx] = util.NewRangesResource("ports", ArrayToRanges(leavingPorts))
					ask := util.NewRangesResource("ports", ArrayToRanges(takingPorts))
					return newResources, ask, true
				}
			}
		}
		return resources, nil, false
	}
	return ret
}
