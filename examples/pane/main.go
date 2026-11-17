// Command pane is a raw-mode program that shows a surface in part of its
// screen, with surf, cells and hottyterm alone: a list of its own on the
// left, a surface with the selected file's details on the right, and the
// keyboard moving between them. On a hotty host the host draws the
// surface; on any other terminal the program draws it in cells.
// surftea/examples/files is the same program on Bubble Tea.
//
//	j, k, arrows   move the list (while the list has the keyboard)
//	tab            give the surface the keyboard; tab past its last
//	               element, or a click on the list, gives it back
//	q, ctrl+c      quit
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/x/ansi"
	"neuroplast.io/hotty-go/hottyterm"
)

func main() {
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, "pane:", err)
		os.Exit(1)
	}
}

func main1() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	t, err := hottyterm.Open("pane")
	if err != nil {
		return err
	}
	defer t.Close()
	restore, err := t.Raw()
	if err != nil {
		return err
	}
	defer restore()
	// Natively nothing reports sizes: the program hears SIGWINCH and tells
	// the Term, whose Sizes the loop reads.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	go func() {
		for range winch {
			t.Resized(t.Size())
		}
	}()
	if err := t.Send(ansi.SetModeAltScreenSaveCursor, ansi.HideCursor, ansi.EraseEntireScreen); err != nil {
		return err
	}
	defer t.Send(ansi.ShowCursor, ansi.ResetModeAltScreenSaveCursor)
	return newProgram(t, NewApp()).run(ctx)
}
