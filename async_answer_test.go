package async_answer

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestA(t *testing.T) {
	const reqKey = "reqKey1"
	r, _ := ApplyReq(reqKey)

	// 模拟异步响应
	go func() {
		time.Sleep(time.Millisecond * 100)
		AnswerReq(reqKey, "ok", nil)
	}()

	v, err := r.WaitAnswer(time.Millisecond * 200)
	if err != nil {
		t.Error("错误", err)
		return
	}
	if v != "ok" {
		t.Errorf("错误 v != ok. is %v", v)
		return
	}
}

func TestB(t *testing.T) {
	const reqKey = "reqKey1"
	r, _ := ApplyReq(reqKey)

	// 模拟异步响应
	testErr := errors.New("test")
	go func() {
		time.Sleep(time.Millisecond * 100)
		AnswerReq(reqKey, nil, testErr)
	}()

	v, err := r.WaitAnswer(time.Millisecond * 200)
	if err != testErr {
		t.Error("错误", err)
		return
	}
	t.Log("结果", v)
}

func TestC(t *testing.T) {
	const reqKey = "reqKey1"
	r, _ := ApplyReq(reqKey)

	// 模拟响应
	AnswerReq(reqKey, "ok", nil)

	v, err := r.WaitAnswer(time.Millisecond * 200)
	if err != nil {
		t.Error("错误", err)
		return
	}
	if v != "ok" {
		t.Errorf("错误 v != ok. is %v", v)
		return
	}
}

func TestTimeout(t *testing.T) {
	const reqKey = "reqKey2"
	r, _ := ApplyReq(reqKey)

	v, err := r.WaitAnswer(time.Millisecond * 200)
	if err != context.DeadlineExceeded {
		t.Errorf("错误. v=%v, err=%v", v, err)
		return
	}
}

func TestRepetitionApply(t *testing.T) {
	const reqKey = "reqKey3"
	_, _ = ApplyReq(reqKey)
	_, ok := ApplyReq(reqKey)
	if ok {
		t.Errorf("错误, 能重复申请")
		return
	}
}

func TestRepetitionCallWaitAnswer(t *testing.T) {
	const reqKey = "reqKey4"
	r, _ := ApplyReq(reqKey)
	_, _ = r.WaitAnswer(time.Millisecond * 200)
	_, err := r.WaitAnswer(time.Millisecond * 200)
	if err != ErrRepetitionCallWaitAnswer {
		t.Errorf("错误, 能重复等待")
		return
	}
}

func TestDelete(t *testing.T) {
	const reqKey = "reqKey5"
	req, _ := ApplyReq(reqKey)
	req.Delete()
	_, ok := ApplyReq(reqKey)
	if !ok {
		t.Errorf("错误, 无法申请")
		return
	}
}
