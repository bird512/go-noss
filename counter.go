package main

import "sync"

type Counter struct {
	val int
	mu  sync.Mutex
}

func (c *Counter) Inc() {
	c.mu.Lock() // 在修改前加锁
	c.val++
	c.mu.Unlock() // 修改后解锁
}

func (c *Counter) Dec() {
	c.mu.Lock() // 在修改前加锁
	c.val--
	c.mu.Unlock() // 修改后解锁
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock() // 使用defer确保在函数退出时解锁
	return c.val
}
