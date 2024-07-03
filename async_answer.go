package async_answer

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ShardCount uint16 = 1 << 8 // 分片数
)

var ErrRepetitionCallWaitAnswer = errors.New("repetition call WaitAnswer")

type AsyncAnswer interface {
	// 申请一个请求
	ApplyReq(key string) (Req, bool)
	// 响应一个请求
	AnswerReq(key string, v interface{}, e error)
}
type Req interface {
	// 等待响应, timeout=0表示永久等待
	WaitAnswer(timeout time.Duration) (interface{}, error)
	// 删除, 如果取消请求可以删除这个req, 如果执行了 WaitAnswer 则这个方法无效
	Delete()
}

type asyncAnswer struct {
	mxs         []*sync.Mutex
	waits       []map[string]*waitData
	andOpValMod uint32
}

type waitData struct {
	v interface{} // 结果
	e error       // 错误

	t *time.Timer
	w sync.WaitGroup
}

// 创建一个AsyncAnswer. shardCount必须为2的幂
func NewAsyncAnswer(shardCount ...uint16) *asyncAnswer {
	count := ShardCount
	if len(shardCount) > 0 && shardCount[0] > 0 {
		count = shardCount[0]
		if count&(count-1) != 0 {
			panic(errors.New("shardCount must power of 2"))
		}
	}

	mxs := make([]*sync.Mutex, count)
	mms := make([]map[string]*waitData, count)
	for i := uint16(0); i < count; i++ {
		mxs[i] = new(sync.Mutex)
		mms[i] = make(map[string]*waitData)
	}
	return &asyncAnswer{
		mxs:         mxs,
		waits:       mms,
		andOpValMod: uint32(count - 1),
	}
}

func (a *asyncAnswer) getShard(key string) (*sync.Mutex, map[string]*waitData) {
	f := fnv.New32a()
	_, _ = f.Write([]byte(key))
	n := f.Sum32()
	shard := n & a.andOpValMod
	return a.mxs[shard], a.waits[shard]
}

// 申请一个请求
func (a *asyncAnswer) ApplyReq(key string) (Req, bool) {
	mx, waits := a.getShard(key)

	mx.Lock()
	defer mx.Unlock()

	_, ok := waits[key]
	if ok { // 已存在
		return nil, false
	}

	w := &waitData{}
	w.w.Add(1)
	waits[key] = w

	req := &reqCli{
		key:   key,
		mx:    mx,
		waits: waits,
		w:     w,
	}
	return req, true
}

type reqCli struct {
	isWait uint32
	key    string
	mx     *sync.Mutex
	waits  map[string]*waitData
	w      *waitData
}

// 等待响应, timeout=0表示永久等待
func (r *reqCli) WaitAnswer(timeout time.Duration) (interface{}, error) {
	if !atomic.CompareAndSwapUint32(&r.isWait, 0, 1) {
		return nil, ErrRepetitionCallWaitAnswer
	}

	if timeout > 0 {
		r.mx.Lock()
		r.w.t = time.AfterFunc(timeout, func() {
			r.mx.Lock()
			defer r.mx.Unlock()

			if r.w.v != nil || r.w.e != nil { // 可能刚好拿到结果. (此函数在下面的 Wait() 之后执行)
				return
			}

			// 超时解锁
			r.w.v = nil
			r.w.e = context.DeadlineExceeded
			r.w.w.Done()
		})
		r.mx.Unlock()
	}

	r.w.w.Wait()

	r.mx.Lock()
	defer r.mx.Unlock()
	delete(r.waits, r.key)
	if r.w.t != nil {
		r.w.t.Stop()
	}

	return r.w.v, r.w.e
}
func (r *reqCli) Delete() {
	if !atomic.CompareAndSwapUint32(&r.isWait, 0, 1) {
		return
	}

	r.mx.Lock()
	delete(r.waits, r.key)
	r.mx.Unlock()
}

// 响应
func (a *asyncAnswer) AnswerReq(key string, v interface{}, e error) {
	mx, waits := a.getShard(key)

	mx.Lock()
	defer mx.Unlock()

	w, ok := waits[key]
	if !ok { // 不存在
		return
	}

	w.v = v
	w.e = e
	w.w.Done()
}

var defAA = NewAsyncAnswer()

// 申请一个请求
func ApplyReq(key string) (Req, bool) {
	return defAA.ApplyReq(key)
}

// 响应一个请求
func AnswerReq(key string, v interface{}, e error) {
	defAA.AnswerReq(key, v, e)
}
