package common

import (
	"errors"
	"fmt"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"math/rand"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func ApplyScalarResources(mutableResources []*mesos.Resource, cpus float64, mem float64, disk float64) []*mesos.Resource {
	mutableResources = ApplyScalarResource(mutableResources, "cpus", cpus)
	mutableResources = ApplyScalarResource(mutableResources, "mem", mem)
	mutableResources = ApplyScalarResource(mutableResources, "disk", disk)
	return mutableResources
}

func ApplyScalarResource(mutableResources []*mesos.Resource, name string, value float64) []*mesos.Resource {
	for _, resource := range util.FilterResources(mutableResources, func(res *mesos.Resource) bool { return res.GetName() == name }) {
		resource.Scalar.Value = &value
	}
	return mutableResources
}

func ScalarResourcesWillFit(immutableResources []*mesos.Resource, cpus float64, mem float64, disk float64) bool {
	if !ScalarResourceWillFit(immutableResources, "cpus", cpus) {
		return false
	}
	if !ScalarResourceWillFit(immutableResources, "mem", mem) {
		return false
	}
	if !ScalarResourceWillFit(immutableResources, "disk", disk) {
		return false
	}

	return true
}

func ScalarResourceWillFit(immutableResources []*mesos.Resource, name string, value float64) bool {
	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.GetName() == name }) {
		if resource.GetScalar().GetValue() < (value) {
			return false
		}
	}
	return true
}

func ResourcesHaveReservations(immutableResources []*mesos.Resource) {
	if common.ResourceHasReservation(immutableResources, "cpus") &&
		common.ResourceHasReservation(immutableResources, "mem") &&
		common.ResourceHasReservation(immutableResources, "disk") {
		return true
	}

	return false
}

func ResourceHasReservation(immutableResources []*mesos.Resource, name string) bool {
	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.GetName() == name }) {
		if name == "disk" {
			if resource.Disk != nil && resource.Reservation != nil {
				return true
			}
		} else if resource.Reservation != nil
			return true
		}
	}
	return false
}

type intarray []int64

func (a intarray) Len() int           { return len(a) }
func (a intarray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a intarray) Less(i, j int) bool { return a[i] < a[j] }

type ResourceAsker (func([]*mesos.Resource) (remaining []*mesos.Resource, ask *mesos.Resource, success bool))

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
					var sliceLoc int
					// Calculate the slice where I'm taking ports:
					if len(ports)-portCount == 0 {
						sliceLoc = 0
					} else {
						sliceLoc = rand.Intn(len(ports) - portCount)
					}
					takingPorts := make([]int64, portCount)
					copy(takingPorts, ports[sliceLoc:(sliceLoc+portCount)])
					leavingPorts := make([]int64, len(ports)-portCount)
					copy(leavingPorts, ports[:sliceLoc])
					copy(leavingPorts[sliceLoc:], ports[(sliceLoc+portCount):])
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

func KillEPMD(dir string) error {
	globs, err := filepath.Glob(filepath.Join(dir, "/erts-*/bin/epmd"))
	if err != nil {
		return err
	}
	if len(globs) != 1 {
		return errors.New(fmt.Sprintf("Not the right number of globs: %d", len(globs)))
	}

	cpEPMDCmd := exec.Command("/bin/cp", "/bin/true", globs[0])
	err = cpEPMDCmd.Run()
	if err != nil {
		return err
	}
	return nil
}
