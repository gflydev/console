package console

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gflydev/core/errors"
	"github.com/gflydev/core/log"
	"github.com/gflydev/core/utils"
	"github.com/hibiken/asynq"
	"sync"
	"time"
)

// ===========================================================================
//                                Task Payload
// ===========================================================================

// TaskPayload wraps asynq.Task and provides helper methods for easier task handling.
type TaskPayload struct {
	// task is the underlying asynq task
	task *asynq.Task
}

// NewCustomTask creates a new TaskPayload wrapper around asynq.Task.
func NewCustomTask(task *asynq.Task) *TaskPayload {
	return &TaskPayload{task: task}
}

// BindPayload unmarshals the JSON payload into the provided struct.
// Example: task.GetPayload(&myStruct)
func (ct *TaskPayload) BindPayload(v interface{}) error {
	return json.Unmarshal(ct.task.Payload(), v)
}

// GetPayload returns the raw payload bytes.
func (ct *TaskPayload) GetPayload() []byte {
	return ct.task.Payload()
}

// GetType returns the task type/name.
func (ct *TaskPayload) GetType() string {
	return ct.task.Type()
}

// GetResultWriter returns the result writer for storing task results.
// Can be used to write results that will be returned to the client.
func (ct *TaskPayload) GetResultWriter() *asynq.ResultWriter {
	return ct.task.ResultWriter()
}

// ===========================================================================
//                                Queue task
// ===========================================================================

// ITask The interface task.
type ITask interface {
	// Dequeue get out and process task in queue.
	Dequeue(task *TaskPayload) error
}

// Task Abstract task.
type Task struct{}

func (t Task) Dequeue(task *TaskPayload) error {
	return errors.NotImplemented
}

// ===========================================================================
//                                Queue handler
// ===========================================================================

func getRedisClientOpt() asynq.RedisClientOpt {
	// Build Redis connection URL.
	redisConnURL := fmt.Sprintf(
		"%s:%d",
		utils.Getenv("REDIS_HOST", "localhost"),
		utils.Getenv("REDIS_PORT", 6379),
	)

	// Define Redis database number.
	dbNumber := utils.Getenv("REDIS_QUEUE_DB", 0)

	return asynq.RedisClientOpt{
		Addr:     redisConnURL,
		Password: utils.Getenv("REDIS_PASSWORD", ""),
		DB:       dbNumber,
	}
}

var (
	client     *asynq.Client
	clientOnce sync.Once
)

// getClient lazily builds the shared asynq client on first use so that the
// Redis options are read after the application has loaded its environment
// (e.g. from a .env file), not at package-import time.
func getClient() *asynq.Client {
	clientOnce.Do(func() {
		client = asynq.NewClient(getRedisClientOpt())
	})
	return client
}

// CloseClient releases the shared asynq client and its Redis connections.
// Call it on application shutdown. Safe to call even if the client was never
// used.
func CloseClient() error {
	if client == nil {
		return nil
	}
	return client.Close()
}

// createTaskHandler creates an adapter function that converts asynq.Task to TaskPayload.
// This allows the ITask interface to work with TaskPayload while asynq expects *asynq.Task.
func createTaskHandler(task ITask) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, asynqTask *asynq.Task) error {
		// Wrap asynq.Task in TaskPayload
		customTask := NewCustomTask(asynqTask)
		// Call the task's Dequeue method with TaskPayload
		return task.Dequeue(customTask)
	}
}

// StartQueueWorker Start queue worker.
// Worker handles a Task(job) was pushed to Queue (Redis) from somewhere.
//   - Many queue Workers run (in a separated server) and try to get Task in Queue to handle
//   - Somewhere in or outside application add a new Task to Queue.
func StartQueueWorker() {
	// Create queue worker
	srv := asynq.NewServer(
		getRedisClientOpt(),
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optional specify multiple queues (Queue type) with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	// mux maps a type to a handler
	mux := asynq.NewServeMux()

	// Register task handlers
	for key, task := range queueTasks {
		// Create adapter to convert asynq.Task to TaskPayload
		handler := createTaskHandler(task)
		mux.HandleFunc(key, handler)
		log.Infof("Init queue task %s", key)
	}

	// Start queue worker
	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}

// queueTasks pool to store task in queue.
var queueTasks = make(map[string]ITask)

// RegisterTask Register a new task to pool.
func RegisterTask(task ITask, name string) {
	queueTasks[name] = task
}

// DispatchTask push a task to queue.
func DispatchTask(data interface{}, name string) {
	startTime := time.Now()

	if err := handleEnqueue(data, name); err != nil {
		log.Errorf("Error %v", err)
	}

	log.Infof("[RUN] Dispatch Task %s - %v", name, time.Since(startTime))
}

func handleEnqueue(data interface{}, name string) error {
	// Get the corresponding task instance to process
	_, ok := queueTasks[name]
	if !ok {
		log.Errorf("Invalid queue `%s`", name)

		return errors.InvalidParameter
	}

	// Encode Payload
	payload, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Encode error %v. Error %v", data, err)

		return err
	}

	// Create new task and push to Queue.
	task := asynq.NewTask(name, payload)

	// Enqueue task
	info, err := getClient().Enqueue(task)
	if err != nil {
		log.Errorf("Could not enqueue task: %v", err)

		return err
	}
	log.Infof("Enqueue task: %v", info)

	return nil
}
