package redis_v2

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"

	"github.com/bitleak/lmstfy/engine"
	"github.com/sirupsen/logrus"
)

func TestTimerV2_Add(t *testing.T) {
	timer, err := NewTimer("timer_set_1", R, time.Second)
	if err != nil {
		panic(fmt.Sprintf("Failed to new timer: %s", err))
	}
	job := engine.NewJob("ns-timer", "q1", []byte("hello msg 1"), 10, 0, 1, 0)
	if err = timer.Add(job.Namespace(), job.Queue(), job.ID(), 10, 1); err != nil {
		t.Errorf("Failed to add job to timer: %s", err)
	}
}

func TestTimerV2_Tick(t *testing.T) {
	timer, err := NewTimer("timer_set_2", R, time.Second)
	if err != nil {
		panic(fmt.Sprintf("Failed to new timer: %s", err))
	}
	defer timer.Shutdown()
	priority := uint8(7)
	job := engine.NewJob("ns-timer", "q2", []byte("hello msg 2"), 5, 0, 1, priority)
	pool := NewPool(R)
	pool.Add(job)
	timer.Add(job.Namespace(), job.Queue(), job.ID(), 3, 1)
	wait := make(chan struct{})
	go func() {
		defer func() {
			wait <- struct{}{}
		}()
		val, err := R.Conn.BZPopMax(3*time.Second, join(QueuePrefix, "ns-timer", "q2")).Result()
		if err != nil && err != redis.Nil {
			t.Fatalf("Failed to pop the job from target queue, err: %s", err.Error())
		}
		if err == redis.Nil {
			t.Fatal("Got non-empty job was expected, but got nothing")
		}
		tries, jobID, err := structUnpack(val.Member.(string))
		if err != nil {
			t.Fatalf("Failed to decode the job pop from queue")
		}
		gotPriority := uint8(int64(val.Score) >> priorityShift)
		if gotPriority != priority {
			t.Fatalf("Mismatch job priority, %d was expected but got %d", priority, gotPriority)
		}
		if tries != 1 || jobID != job.ID() {
			t.Fatal("Job data mismatched")
		}
	}()
	<-wait
}

func BenchmarkTimerV2(b *testing.B) {
	// Disable logging temporarily
	logger.SetLevel(logrus.ErrorLevel)
	defer logger.SetLevel(logrus.DebugLevel)

	t, err := NewTimer("timer_set_3", R, time.Second)
	if err != nil {
		panic(fmt.Sprintf("Failed to new timer: %s", err))
	}
	defer t.Shutdown()
	b.Run("Add", benchmarkTimerV2_Add(t))

	b.Run("Pop", benchmarkTimerV2_Pop(t))
}

func benchmarkTimerV2_Add(timer *Timer) func(b *testing.B) {
	pool := NewPool(R)
	return func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			job := engine.NewJob("ns-timer", "q3", []byte("hello msg 1"), 100, 0, 1, 0)
			pool.Add(job)
			timer.Add(job.Namespace(), job.Queue(), job.ID(), 1, 1)
		}
	}
}

func benchmarkTimerV2_Pop(timer *Timer) func(b *testing.B) {
	return func(b *testing.B) {
		key := join(QueuePrefix, "ns-timer", "q3")
		b.StopTimer()
		pool := NewPool(R)
		for i := 0; i < b.N; i++ {
			job := engine.NewJob("ns-timer", "q3", []byte("hello msg 1"), 100, 0, 1, 0)
			pool.Add(job)
			timer.Add(job.Namespace(), job.Queue(), job.ID(), 1, 1)
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			R.Conn.BRPop(5*time.Second, key)
		}
	}
}

// How long did it take to fire 10000 due jobs
func BenchmarkTimerV2_Pump(b *testing.B) {
	// Disable logging temporarily
	logger.SetLevel(logrus.ErrorLevel)
	defer logger.SetLevel(logrus.DebugLevel)

	b.StopTimer()

	pool := NewPool(R)
	timer, err := NewTimer("timer_set_4", R, time.Second)
	if err != nil {
		panic(fmt.Sprintf("Failed to new timer: %s", err))
	}
	timer.Shutdown()
	for i := 0; i < 10000; i++ {
		job := engine.NewJob("ns-timer", "q4", []byte("hello msg 1"), 100, 0, 1, 0)
		pool.Add(job)
		timer.Add(job.Namespace(), job.Queue(), job.ID(), 1, 1)
	}

	b.StartTimer()
	timer.pump(time.Now().Unix() + 1)
}
