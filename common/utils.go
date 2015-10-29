package common

import (
	"errors"
	"fmt"
	mesos "github.com/basho-labs/mesos-go/mesosproto"
	util "github.com/basho-labs/mesos-go/mesosutil"
	"math/rand"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func FilterReservedResources(immutableResources []*mesos.Resource) []*mesos.Resource {
	reserved := []*mesos.Resource{}
	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.Reservation != nil }) {
		reserved = append(reserved, resource)
	}
	return reserved
}

func FilterUnreservedResources(immutableResources []*mesos.Resource) []*mesos.Resource {
	unreserved := []*mesos.Resource{}
	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.Reservation == nil }) {
		unreserved = append(unreserved, resource)
	}
	return unreserved
}

func ApplyScalarResources(mutableResources []*mesos.Resource, cpus float64, mem float64, disk float64) []*mesos.Resource {
	mutableResources = ApplyScalarResource(mutableResources, "cpus", cpus)
	mutableResources = ApplyScalarResource(mutableResources, "mem", mem)
	mutableResources = ApplyScalarResource(mutableResources, "disk", disk)
	return mutableResources
}

func ApplyRangesResource(mutableResources []*mesos.Resource, portCount int) ([]*mesos.Resource, *mesos.Resource) {
	leftoverPorts, extractedPorts := CreatePortsResourceFromResources(mutableResources, portCount)

	for idx := range util.FilterResources(mutableResources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
		mutableResources[idx] = leftoverPorts
	}
	return mutableResources, extractedPorts
}

func ApplyScalarResource(mutableResources []*mesos.Resource, name string, value float64) []*mesos.Resource {
	for _, resource := range util.FilterResources(mutableResources, func(res *mesos.Resource) bool { return res.GetName() == name }) {
		newValue := *resource.Scalar.Value - value
		resource.Scalar.Value = &newValue
	}
	return mutableResources
}

func PortResourceWillFit(immutableResources []*mesos.Resource, portCount int) bool {
	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
		ports := RangesToArray(resource.GetRanges().GetRange())
		if len(ports) < portCount {
			return false
		}
	}
	return true
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

func CreatePortsResourceFromResources(immutableResources []*mesos.Resource, portCount int) (*mesos.Resource, *mesos.Resource) {
	leftoverPortsResource := &mesos.Resource{}
	ask := &mesos.Resource{}

	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
		ports := RangesToArray(resource.GetRanges().GetRange())

		sliceLoc := 0
		if len(ports)-portCount > 0 {
			sliceLoc = rand.Intn(len(ports) - portCount)
		}
		takingPorts := make([]int64, portCount)
		copy(takingPorts, ports[sliceLoc:(sliceLoc+portCount)])
		leavingPorts := make([]int64, len(ports)-portCount)
		copy(leavingPorts, ports[:sliceLoc])
		copy(leavingPorts[sliceLoc:], ports[(sliceLoc+portCount):])
		leftoverPortsResource = util.NewRangesResource("ports", ArrayToRanges(leavingPorts))
		ask := util.NewRangesResource("ports", ArrayToRanges(takingPorts))
		return leftoverPortsResource, ask
	}

	return leftoverPortsResource, ask
}

func RemoveReservations(resources []*mesos.Resource) []*mesos.Resource {
	for _, resource := range resources {
		resource.Reservation = nil
		resource.Disk = nil
		resource.Role = nil
	}

	return resources
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
