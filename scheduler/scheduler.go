package main
import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"github.com/gogo/protobuf/proto"
	"log"
	"net"
	"strconv"
	"io"
	"os"
	"github.com/kr/pretty"
    "fmt"

)

var (
	CPUS_PER_TASK=3.0
	MEM_PER_TASK=10.0
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)
type ExampleScheduler struct {
	executor      *mesos.ExecutorInfo
	tasksLaunched int
	tasksFinished int
	totalTasks    int
}
func newExampleScheduler(exec *mesos.ExecutorInfo) *ExampleScheduler {
	total := 5
	return &ExampleScheduler{
		executor:      exec,
		tasksLaunched: 0,
		tasksFinished: 0,
		totalTasks:    total,
	}
}
func (sched *ExampleScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	Info.Println("Framework Registered with Master ", masterInfo)
}

func (sched *ExampleScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	Info.Println("Framework Re-Registered with Master ", masterInfo)
}

func (sched *ExampleScheduler) Disconnected(sched.SchedulerDriver) {}

func (sched *ExampleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {

	for _, offer := range offers {

		if 1==2 {
			fmt.Printf("%# v", pretty.Formatter(offer))
		}
		cpuResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "cpus"
		})
		cpus := 0.0
		for _, res := range cpuResources {
			cpus += res.GetScalar().GetValue()
		}

		memResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "mem"
		})
		mems := 0.0
		for _, res := range memResources {
			mems += res.GetScalar().GetValue()
		}

		Info.Println("Received Offer <", offer.Id.GetValue(), "> with cpus=", cpus, " mem=", mems)

		remainingCpus := cpus
		remainingMems := mems

		var tasks []*mesos.TaskInfo
		for sched.tasksLaunched < sched.totalTasks &&
		CPUS_PER_TASK <= remainingCpus &&
		MEM_PER_TASK <= remainingMems {

			sched.tasksLaunched++

			taskId := &mesos.TaskID{
				Value: proto.String(strconv.Itoa(sched.tasksLaunched)),
			}

			task := &mesos.TaskInfo{
				Name:     proto.String("go-task-" + taskId.GetValue()),
				TaskId:   taskId,
				SlaveId:  offer.SlaveId,
				Executor: sched.executor,
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", CPUS_PER_TASK),
					util.NewScalarResource("mem", MEM_PER_TASK),
					util.NewRangesResource("ports", []*mesos.Value_Range{util.NewValueRange(31005, 31010)}),
				},
			}
			Info.Printf("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			tasks = append(tasks, task)
			remainingCpus -= CPUS_PER_TASK
			remainingMems -= MEM_PER_TASK
		}
		Info.Println("Launching ", len(tasks), "tasks for offer", offer.Id.GetValue())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *ExampleScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	Info.Println("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())
	if status.GetState() == mesos.TaskState_TASK_FINISHED {
		sched.tasksFinished++
	}

	if sched.tasksFinished >= sched.totalTasks {
		Info.Println("Total tasks completed, stopping framework.")
		driver.Stop(false)
	}

	if status.GetState() == mesos.TaskState_TASK_LOST ||
	status.GetState() == mesos.TaskState_TASK_KILLED ||
	status.GetState() == mesos.TaskState_TASK_FAILED {
		Info.Println(
			"Aborting because task", status.TaskId.GetValue(),
			"is in unexpected state", status.State.String(),
			"with message", status.GetMessage(),
		)
		driver.Abort()
	}
}

func (sched *ExampleScheduler) OfferRescinded(sched.SchedulerDriver, *mesos.OfferID) {}

func (sched *ExampleScheduler) FrameworkMessage(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, string) {
}
func (sched *ExampleScheduler) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {}
func (sched *ExampleScheduler) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
}

func (sched *ExampleScheduler) Error(driver sched.SchedulerDriver, err string) {
	Info.Println("Scheduler received error:", err)
}

func parseIP(address string) net.IP {
	addr, err := net.LookupIP(address)
	if err != nil {
		log.Fatal(err)
	}
	if len(addr) < 1 {
		log.Fatalf("failed to parse IP from address '%v'", address)
	}
	return addr[0]
}
func setupLogger(traceHandle io.Writer,
infoHandle io.Writer,
warningHandle io.Writer,
errorHandle io.Writer) {
	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}
func main() {
	setupLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	exec := mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Name:       proto.String("Test Executor (Go)"),
		Source:     proto.String("go_test"),
		Command: &mesos.CommandInfo{
			Value: proto.String("/bin/sleep 10000"),
			//Uris:  executorUris,
		},
	}
	fwinfo := &mesos.FrameworkInfo{
		User: proto.String("sargun"), // Mesos-go will fill in user.
		Name: proto.String("Test Framework (Go)"),
	}
	cred := (*mesos.Credential)(nil)
	bindingAddress := parseIP("33.33.33.1")
	config := sched.DriverConfig{
		Scheduler:      newExampleScheduler(&exec),
		Framework:      fwinfo,
		Master:         "33.33.33.2:5050",
		Credential:     cred,
		BindingAddress: bindingAddress,
	//	WithAuthContext: func(ctx context.Context) context.Context {
	//		ctx = auth.WithLoginProvider(ctx, *authProvider)
	//		ctx = sasl.WithBindingAddress(ctx, bindingAddress)
	//		return ctx
	//	},
	}
	driver, err := sched.NewMesosSchedulerDriver(config)

	if err != nil {
		Error.Println("Unable to create a SchedulerDriver ", err.Error())
	}

	if stat, err := driver.Run(); err != nil {
		Info.Printf("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}