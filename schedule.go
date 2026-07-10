package console

import (
	"github.com/gflydev/core/log"
	"github.com/gflydev/core/utils"
	"github.com/robfig/cron/v3"
)

// IJob The interface task.
type IJob interface {
	GetTime() string
	Handle()
}

// Job pool
var jobs []IJob

// RegisterJob Register a new job to pool.
func RegisterJob(job IJob) {
	jobs = append(jobs, job)
}

func StartScheduler() {
	c := cron.New(cron.WithSeconds())

	// Define the Cron job schedule
	for _, job := range jobs {
		_, err := c.AddFunc(job.GetTime(), job.Handle)
		if err != nil {
			// Skip the faulty job (e.g. invalid cron spec) but keep
			// registering the rest instead of aborting the scheduler.
			log.Errorf("Could not register job %s with spec %q: %v",
				utils.ReflectType(job), job.GetTime(), err)
			continue
		}
		log.Infof("Init schedule job %s", utils.ReflectType(job))
	}

	// Start the Cron job scheduler
	c.Start()

	// Run forever
	var forever chan struct{}
	<-forever
}
