// Package utils gathers utility functionality not part of the main indexing process
package utils

import (
	"fmt"
	"log/slog"
	"os/exec"
)

func Notify(body string, persistent bool) {
	programName := "Igloo"

	var timeout int32
	switch persistent {
	case true:
		timeout = 0
	case false:
		timeout = 5000
	}

	cmd := exec.Command("dbus-send",
		"--session",
		"--dest=org.freedesktop.Notifications",
		"--type=method_call",
		"--print-reply",
		"/org/freedesktop/Notifications",
		"org.freedesktop.Notifications.Notify",
		fmt.Sprintf("string:%s", programName), // program name for daemon grouping
		"uint32:0",                            // replaces_id (0 = new bubble)
		"string:info",                         // path to .png or standard icon name
		fmt.Sprintf("string:%s", programName), // summary/title
		fmt.Sprintf("string:%s", body),        // body/notification text
		"array:string:",                       // actions for drawing buttons et.c.(ex: string:"yes","Accept","no","Ignore")
		"dict:string:variant:",                // hints (ex: variant:urgency,byte:0)
		fmt.Sprintf("int32:%d", timeout),      // expire_timeout (0 = stays until user clicks the notif)
	)

	if err := cmd.Run(); err != nil {
		slog.Error(fmt.Sprintf("notification for cmd %s", cmd), "call", "cmd.Run()", "err", err)
	}
}
