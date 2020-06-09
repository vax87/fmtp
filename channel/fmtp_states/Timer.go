package fmtp_states

import (
	"context"
	//"fmt"
	//"runtime"
	"sync"
	"time"

	"fdps/fmtp/fmtp"
)

// таймер с возможностью запуска/остановки.
// по таймауту генерит fmtp событие
type Timer struct {
	duration    time.Duration       // интервал срабатывания таймера
	timer       *time.Timer         // таймер
	fmtpEvent   fmtp.FmtpEvent      // событие по таймауту
	eventChan   chan fmtp.FmtpEvent // канал для передачи события
	timerCtx    context.Context
	timerCancel context.CancelFunc
}

func (ft *Timer) startTimer() {
	//fmt.Println("\tStart Timer ", ft.fmtpEvent.ToString(), time.Now().Format("15:04:05.333"))
	go func() {
		ft.timerCtx, ft.timerCancel = context.WithTimeout(context.Background(), ft.duration)

		<-ft.timerCtx.Done()
		//fmt.Println("context error ", ft.fmtpEvent.ToString(), ft.timerCtx.Err(), time.Now().Format("15:04:05.333"))
		if ft.timerCtx.Err() == context.DeadlineExceeded {
			ft.eventChan <- ft.fmtpEvent
			//fmt.Println("\tTimeout ", ft.fmtpEvent.ToString(), time.Now().Format("15:04:05.333"))
		}
	}()
}

func (ft *Timer) stopTimer() {
	//fmt.Println("\tStop Timer ", ft.fmtpEvent.ToString(), time.Now().Format("15:04:05.333"))
	go func() {
		if ft.timerCancel != nil {
			ft.timerCancel()
		}
	}()
}

func (ft *Timer) restartTimer() {
	//fmt.Println("routines count ", runtime.NumGoroutine())
	//fmt.Println("\tRestart Timer ", ft.fmtpEvent.ToString(), time.Now().Format("15:04:05.333"))
	var waitGr sync.WaitGroup
	waitGr.Add(1)
	go func() {
		defer waitGr.Done()
		if ft.timerCancel != nil {
			ft.timerCancel()
		}

	}()
	waitGr.Wait()
	ft.startTimer()
}

func newFmtpTimer(curDuration time.Duration, curFmtpEvent fmtp.FmtpEvent) *Timer {
	return &Timer{duration: curDuration,
		timer:     time.NewTimer(curDuration),
		fmtpEvent: curFmtpEvent,
		eventChan: make(chan fmtp.FmtpEvent),
	}
}
