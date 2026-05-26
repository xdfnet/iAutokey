//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework ApplicationServices -framework Foundation

#include <ApplicationServices/ApplicationServices.h>
#include <dispatch/dispatch.h>
#include <stdlib.h>

#define EVENT_SOURCE_STATE_PRIVATE ((CGEventSourceStateID)-1)

static CGEventSourceRef gEventSource = NULL;

typedef struct {
	int64_t      keyCode;
	CGEventFlags flagMask;
	int          wasPressed;
	int64_t      delayNs;
} auto_enter_state_t;

CGEventRef auto_enter_callback(CGEventTapProxy proxy, CGEventType type,
                               CGEventRef event, void *refcon) {
	if (type != kCGEventFlagsChanged) return event;

	auto_enter_state_t *s = (auto_enter_state_t *)refcon;
	if (CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode) != s->keyCode) {
		return event;
	}

	CGEventFlags flags = CGEventGetFlags(event);

	if (flags & s->flagMask) {
		s->wasPressed = 1;
	} else if (s->wasPressed) {
		s->wasPressed = 0;
		int64_t delay = s->delayNs;

		if (!gEventSource) {
			gEventSource = CGEventSourceCreate(EVENT_SOURCE_STATE_PRIVATE);
		}

		dispatch_after(dispatch_time(DISPATCH_TIME_NOW, delay),
			dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
				CGEventRef down = CGEventCreateKeyboardEvent(gEventSource, (CGKeyCode)36, true);
				CGEventRef up   = CGEventCreateKeyboardEvent(gEventSource, (CGKeyCode)36, false);
				CGEventPost(kCGSessionEventTap, down);
				CGEventPost(kCGSessionEventTap, up);
				usleep(1000);
				CFRelease(down);
				CFRelease(up);
			});
	}

	return event;
}
*/
import "C"
import (
	"log"
	"runtime"
	"time"
	"unsafe"
)

func hasAccessibilityPermission() bool {
	return C.AXIsProcessTrusted() != 0
}

var keyMap = map[string]struct {
	keyCode  int64
	flagMask C.CGEventFlags
}{
	"right_command": {0x36, C.kCGEventFlagMaskCommand},
	"left_command":  {0x37, C.kCGEventFlagMaskCommand},
	"right_option":  {0x3D, C.kCGEventFlagMaskAlternate},
	"left_option":   {0x3A, C.kCGEventFlagMaskAlternate},
	"right_shift":   {0x3C, C.kCGEventFlagMaskShift},
	"left_shift":    {0x38, C.kCGEventFlagMaskShift},
	"right_control": {0x3E, C.kCGEventFlagMaskControl},
	"left_control":  {0x3B, C.kCGEventFlagMaskControl},
	"fn":            {0x3F, C.kCGEventFlagMaskSecondaryFn},
}

func startAutoEnter(keyName string, delayMs int) {
	entry, ok := keyMap[keyName]
	if !ok {
		log.Printf("不支持的按键 %q", keyName)
		return
	}
	if delayMs <= 0 {
		delayMs = 500
	}

	state := (*C.auto_enter_state_t)(C.calloc(1, C.sizeof_auto_enter_state_t))
	state.keyCode = C.int64_t(entry.keyCode)
	state.flagMask = entry.flagMask
	state.delayNs = C.int64_t(delayMs) * 1000000

	runtime.LockOSThread()

	eventMask := C.CGEventMask(uint64(1) << C.kCGEventFlagsChanged)
	var tap unsafe.Pointer
	for i := 0; i < 5; i++ {
		tap = unsafe.Pointer(C.CGEventTapCreate(
			C.kCGSessionEventTap,
			C.kCGHeadInsertEventTap,
			C.kCGEventTapOptionDefault,
			eventMask,
			C.CGEventTapCallBack(C.auto_enter_callback),
			unsafe.Pointer(state),
		))
		if tap != nil {
			break
		}
		log.Printf("第 %d 次创建 event tap 失败，重试中...", i+1)
		time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
	}

	if tap == nil {
		C.free(unsafe.Pointer(state))
		runtime.UnlockOSThread()
		log.Printf("创建 event tap 失败，请在 系统设置→隐私与安全性→辅助功能 中允许 iautokey")
		return
	}

	runLoopSource := C.CFMachPortCreateRunLoopSource(C.kCFAllocatorDefault, C.CFMachPortRef(tap), 0)
	C.CFRunLoopAddSource(C.CFRunLoopGetCurrent(), runLoopSource, C.kCFRunLoopCommonModes)
	C.CGEventTapEnable(C.CFMachPortRef(tap), true)

	log.Printf("已启动，按键=%s delay=%dms", keyName, delayMs)
	C.CFRunLoopRun()
}
