package game

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"gunmayhem-go/internal/updater"
)

var mainUpdateNoticeRect = Rect{X: 20, Y: 536, W: 500, H: 44}

const (
	updatePhaseIdle = iota
	updatePhaseChecking
	updatePhaseAvailable
	updatePhaseDownloading
	updatePhaseFailed
)

type updateCheckResult struct {
	release   updater.Release
	available bool
	err       error
}

type updateDownloadResult struct {
	path string
	err  error
}

type updateProgress struct {
	done  int64
	total int64
}

type updateRuntimeState struct {
	phase       int
	release     updater.Release
	checkCh     chan updateCheckResult
	downloadCh  chan updateDownloadResult
	progressCh  chan updateProgress
	done        int64
	total       int64
	message     string
	manualCheck bool
}

func (g *Game) startUpdateCheck(manual bool) {
	if g == nil || g.updateRuntime.phase == updatePhaseDownloading || g.updateRuntime.checkCh != nil {
		return
	}
	if !updater.IsReleaseBuild() {
		if manual {
			g.updateRuntime.message = "DEV BUILD - UPDATE CHECKS ARE ENABLED IN TAGGED RELEASES"
		}
		return
	}
	g.updateRuntime.phase = updatePhaseChecking
	g.updateRuntime.manualCheck = manual
	g.updateRuntime.message = ""
	g.updateRuntime.checkCh = make(chan updateCheckResult, 1)
	ch := g.updateRuntime.checkCh
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		release, available, err := updater.CheckLatest(ctx)
		ch <- updateCheckResult{release: release, available: available, err: err}
	}()
}

func (g *Game) startAutomaticUpdateCheck() {
	if g == nil || g.updateRuntime.phase != updatePhaseIdle {
		return
	}
	g.startUpdateCheck(false)
}

func (g *Game) beginManualUpdateCheck() {
	g.startUpdateCheck(true)
}

func (g *Game) updateUpdaterRuntime() (terminate bool) {
	if g == nil {
		return false
	}
	if ch := g.updateRuntime.checkCh; ch != nil {
		select {
		case result := <-ch:
			g.updateRuntime.checkCh = nil
			manual := g.updateRuntime.manualCheck
			g.updateRuntime.manualCheck = false
			if result.err != nil {
				// Automatic startup checks stay quiet while offline. Manual checks
				// report the failure inside Settings instead of creating a popup.
				g.updateRuntime.phase = updatePhaseIdle
				if manual {
					g.updateRuntime.message = "CHECK FAILED - TRY AGAIN"
				} else {
					g.updateRuntime.message = ""
				}
			} else if result.available {
				g.updateRuntime.phase = updatePhaseAvailable
				g.updateRuntime.release = result.release
				g.updateRuntime.message = "UPDATE AVAILABLE: " + result.release.Tag
			} else {
				g.updateRuntime.phase = updatePhaseIdle
				g.updateRuntime.release = updater.Release{}
				if manual {
					g.updateRuntime.message = "YOU ARE UP TO DATE"
				} else {
					g.updateRuntime.message = ""
				}
			}
		default:
		}
	}

	if ch := g.updateRuntime.progressCh; ch != nil {
		for {
			select {
			case p := <-ch:
				g.updateRuntime.done, g.updateRuntime.total = p.done, p.total
			default:
				goto progressDone
			}
		}
	}
progressDone:

	if ch := g.updateRuntime.downloadCh; ch != nil {
		select {
		case result := <-ch:
			g.updateRuntime.downloadCh = nil
			g.updateRuntime.progressCh = nil
			if result.err != nil {
				g.updateRuntime.phase = updatePhaseFailed
				g.updateRuntime.message = result.err.Error()
				return false
			}
			if err := updater.LaunchApply(result.path); err != nil {
				g.updateRuntime.phase = updatePhaseFailed
				g.updateRuntime.message = err.Error()
				return false
			}
			// The helper is now waiting for this PID to exit. Returning Ebiten's
			// regular termination lets main save config/progress before process exit.
			return true
		default:
		}
	}
	return false
}

func (g *Game) beginUpdateDownload() {
	if g == nil || (g.updateRuntime.phase != updatePhaseAvailable && g.updateRuntime.phase != updatePhaseFailed) {
		return
	}
	release := g.updateRuntime.release
	if release.Asset.URL == "" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		g.updateRuntime.phase = updatePhaseFailed
		g.updateRuntime.message = err.Error()
		return
	}
	updateDir := filepath.Join(filepath.Dir(exe), "data", "update")
	g.updateRuntime.phase = updatePhaseDownloading
	g.updateRuntime.done = 0
	g.updateRuntime.total = release.Asset.Size
	g.updateRuntime.message = ""
	g.updateRuntime.downloadCh = make(chan updateDownloadResult, 1)
	g.updateRuntime.progressCh = make(chan updateProgress, 16)
	downloadCh := g.updateRuntime.downloadCh
	progressCh := g.updateRuntime.progressCh
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		path, err := updater.Download(ctx, release, updateDir, func(done, total int64) {
			select {
			case progressCh <- updateProgress{done: done, total: total}:
			default:
			}
		})
		downloadCh <- updateDownloadResult{path: path, err: err}
	}()
}

func currentUpdateVersionLabel() string {
	if updater.IsReleaseBuild() {
		return updater.CurrentVersion
	}
	return "DEV BUILD"
}

func (g *Game) settingsUpdateStatus() string {
	if g == nil {
		return ""
	}
	switch g.updateRuntime.phase {
	case updatePhaseChecking:
		return "CHECKING GITHUB..."
	case updatePhaseAvailable:
		return "NEW VERSION: " + g.updateRuntime.release.Tag
	case updatePhaseDownloading:
		if g.updateRuntime.total > 0 {
			pct := int(float64(g.updateRuntime.done) * 100 / float64(g.updateRuntime.total))
			if pct < 0 {
				pct = 0
			}
			if pct > 100 {
				pct = 100
			}
			return fmt.Sprintf("DOWNLOADING %s - %d%%", g.updateRuntime.release.Tag, pct)
		}
		return "DOWNLOADING " + g.updateRuntime.release.Tag
	case updatePhaseFailed:
		return "UPDATE FAILED - TRY AGAIN"
	default:
		if g.updateRuntime.message != "" {
			return g.updateRuntime.message
		}
		if !updater.IsReleaseBuild() {
			return "TAGGED RELEASES CAN UPDATE AUTOMATICALLY"
		}
		return "READY TO CHECK"
	}
}

func (g *Game) settingsUpdateActionAvailable() bool {
	return g != nil && (g.updateRuntime.phase == updatePhaseAvailable || (g.updateRuntime.phase == updatePhaseFailed && g.updateRuntime.release.Asset.URL != ""))
}

func (g *Game) updateNoticeVisible() bool {
	if g == nil {
		return false
	}
	switch g.updateRuntime.phase {
	case updatePhaseAvailable, updatePhaseDownloading, updatePhaseFailed:
		return true
	default:
		return false
	}
}

func (g *Game) updateNoticeLabel() string {
	if g == nil {
		return ""
	}
	tag := g.updateRuntime.release.Tag
	switch g.updateRuntime.phase {
	case updatePhaseAvailable:
		return "UPDATE AVAILABLE " + tag + "  -  DOWNLOAD & UPDATE"
	case updatePhaseDownloading:
		if g.updateRuntime.total > 0 {
			pct := int(float64(g.updateRuntime.done) * 100 / float64(g.updateRuntime.total))
			if pct < 0 {
				pct = 0
			}
			if pct > 100 {
				pct = 100
			}
			return fmt.Sprintf("DOWNLOADING UPDATE %s  %d%%", tag, pct)
		}
		return "DOWNLOADING UPDATE " + tag
	case updatePhaseFailed:
		return "UPDATE FAILED - CLICK TO RETRY"
	default:
		return ""
	}
}

func (g *Game) drawUpdateNotice(screen *ebiten.Image) {
	if screen == nil || !g.updateNoticeVisible() {
		return
	}
	r := mainUpdateNoticeRect
	border := color.NRGBA{R: 0xff, G: 0x98, B: 0x00, A: 0xff}
	fill := color.NRGBA{R: 0x1d, G: 0x21, B: 0x28, A: 0xf4}
	if g.updateRuntime.phase == updatePhaseFailed {
		border = color.NRGBA{R: 0xd6, G: 0x55, B: 0x43, A: 0xff}
	}
	ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, border)
	ebitenutil.DrawRect(screen, r.X+2, r.Y+2, r.W-4, r.H-4, fill)
	if g.updateRuntime.phase == updatePhaseDownloading && g.updateRuntime.total > 0 {
		frac := float64(g.updateRuntime.done) / float64(g.updateRuntime.total)
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		ebitenutil.DrawRect(screen, r.X+2, r.Y+r.H-6, (r.W-4)*frac, 4, border)
	}
	drawSourceMenuText(screen, g.updateNoticeLabel(), menuFontCondensedExtraBold, 15.5, color.White, r.X+12, r.Y+12)
}
