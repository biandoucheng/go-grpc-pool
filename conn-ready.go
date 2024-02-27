package gogrpcpool

// 标记为就绪状态
// 1. 这意味着连接被推入就绪通道，且暂时还没有被取用
// 2. 当连接从就绪通道被取用之后，需要调用unsetReady来移除就绪状态
func (c *Conn) setReady() {
	c.readying = true
}

// 移除就绪状态
func (c *Conn) unsetReady() {
	c.readying = false
}
