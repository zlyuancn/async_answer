
一个等待异步响应的工具. 挂起请求直到收到了异步响应. 可用于MQTT请求响应模式.

使用示例

```go
const reqKey = "reqKey"
r, _ := async_answer.ApplyReq(reqKey)

// 申请req后可以开始请求

// 此处模拟异步响应
go func() {
    time.Sleep(time.Millisecond * 100)
    async_answer.AnswerReq(reqKey, "ok", nil)
}()

// 等待异步响应结果
v, err := r.WaitAnswer(time.Millisecond * 200)
if err != nil {
    panic(err)
}
fmt.Println(v)
```
