package game

import (
	"fmt"
	"image/color"
	"strings"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
)

// Online quality-of-life helpers: clipboard copy/paste of join codes,
// asynchronous joins with cancel, run-while-unfocused during online sessions,
// and multi-line error rendering.

const inviteCopyFeedbackDuration = 2500 * time.Millisecond

// customV2JoinCodeBarRect is the status bar on the Custom Game SLOTS page that
// shows the host's invite code. Clicking it copies the code.
var customV2JoinCodeBarRect = Rect{X: 20, Y: 501, W: 860, H: 20}

// ---------------------------------------------------------------------------
// Focus policy

// updateUnfocusedRunPolicy keeps the game loop running while unfocused only
// during online sessions (or while a join is in progress). A host that alt-tabs
// to share the invite code must keep sending snapshots, and a client must keep
// its connection serviced. Local games still pause when the window loses focus,
// as main.go configures at startup.
func (g *Game) updateUnfocusedRunPolicy() {
	if g == nil {
		return
	}
	want := g.netplay != nil || g.pendingNetJoin != nil
	if want == g.runnableUnfocused {
		return
	}
	ebiten.SetRunnableOnUnfocused(want)
	g.runnableUnfocused = want
}

// netClientInputForFocus releases every control while the client window is
// unfocused. The host only keeps the latest input it received, so without this
// an alt-tab while holding a key would keep that key held on the host.
func netClientInputForFocus(input PlayerInputState, focused bool) PlayerInputState {
	if !focused {
		return PlayerInputState{}
	}
	return input
}

// ---------------------------------------------------------------------------
// Clipboard: join input paste and host invite copy

func isClipboardModifierPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
}

func isPasteShortcutJustPressed() bool {
	if isClipboardModifierPressed() && inpututil.IsKeyJustPressed(ebiten.KeyV) {
		return true
	}
	return ebiten.IsKeyPressed(ebiten.KeyShift) && inpututil.IsKeyJustPressed(ebiten.KeyInsert)
}

func trimJoinTokenPunctuation(token string) string {
	return strings.TrimFunc(token, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '[' || r == ']')
	})
}

// normalizePastedJoinInput turns pasted text into something the join field
// accepts. It finds an invite code (GMT1-/GM16-/GM-) or a trycloudflare.com URL
// inside a longer chat message, strips whitespace/quotes and characters the
// field does not accept, and caps the length.
func normalizePastedJoinInput(raw string) string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("\"'`<>(){}“”‘’,;", r)
	})
	if len(fields) == 0 {
		return ""
	}
	pick := ""
	// A trycloudflare URL first: its random hostname may itself contain "gm-".
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), cloudflareTunnelHostSuffix) {
			if wsURL, err := normalizeCloudflareTunnelURL(trimJoinTokenPunctuation(field)); err == nil {
				// Keep only the hostname: it fits the field's character set
				// and the join dispatcher recognises the suffix.
				pick = strings.TrimSuffix(strings.TrimPrefix(wsURL, "wss://"), cloudflareNetplayPath)
				break
			}
		}
	}
	// Most specific prefix first: GMT1- and GM16- do not contain "GM-". The
	// prefix must start the word or follow a separator such as "code:".
	for _, prefix := range []string{cloudflareInviteCodePrefix, inviteCodePrefix, shortOpenCodePrefix} {
		if pick != "" {
			break
		}
		for _, field := range fields {
			upper := strings.ToUpper(field)
			i := strings.Index(upper, prefix)
			if i < 0 {
				continue
			}
			if i > 0 {
				prev := rune(upper[i-1])
				if prev >= 'A' && prev <= 'Z' || prev >= '0' && prev <= '9' || prev == '-' || prev == '.' {
					continue
				}
			}
			pick = strings.ToUpper(trimJoinTokenPunctuation(field[i:]))
			break
		}
	}
	if pick == "" {
		if len(fields) != 1 {
			// Unknown multi-word text: do not guess which word is the address.
			pick = strings.Join(fields, "")
		} else {
			pick = trimJoinTokenPunctuation(fields[0])
		}
	}
	var b strings.Builder
	for _, r := range pick {
		if validNetAddressRune(r) {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > cloudflareTunnelURLMaxLen {
		out = out[:cloudflareTunnelURLMaxLen]
	}
	return out
}

// pasteIntoMultiplayerAddress replaces the join field with clipboard text.
func (g *Game) pasteIntoMultiplayerAddress() {
	text, err := readClipboardText()
	if err != nil {
		g.multiplayerMessage = "PASTE FAILED: " + strings.ToUpper(err.Error())
		networkDiagnostic("JOIN paste failed err=%v", err)
		return
	}
	value := normalizePastedJoinInput(text)
	if value == "" {
		g.multiplayerMessage = "CLIPBOARD HAS NO JOIN CODE"
		return
	}
	g.multiplayerAddress = value
	g.multiplayerFocus = true
	g.multiplayerMessage = "PASTED - PRESS ENTER OR JOIN"
}

// updateInviteCodeClipboard lets the host copy the invite code with Ctrl+C (any
// non-gameplay screen) or by clicking the code bar on the SLOTS page.
func (g *Game) updateInviteCodeClipboard() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || g.screen == screenGameplay {
		return
	}
	copyRequested := isClipboardModifierPressed() && inpututil.IsKeyJustPressed(ebiten.KeyC)
	if !copyRequested && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) &&
		g.screen == screenCustomGame && !g.customNetworkOpen && g.customPage == customPagePlayers {
		mx, my := g.cursorPos()
		copyRequested = customV2JoinCodeBarRect.Contains(float64(mx), float64(my))
	}
	if copyRequested {
		g.copyHostInviteCode()
	}
}

func (g *Game) setInviteCopyFeedback(text string) {
	g.inviteCopyFeedback = text
	g.inviteCopyFeedbackUntil = time.Now().Add(inviteCopyFeedbackDuration)
}

func (g *Game) copyHostInviteCode() bool {
	code, ready := g.currentHostDisplayJoinCode()
	if !ready || code == "" {
		if g.onlineSessionVisibility == onlineSessionInviteOnly {
			g.setInviteCopyFeedback("CODE NOT READY YET")
		} else {
			g.setInviteCopyFeedback("OPEN SESSION - NO CODE TO COPY")
		}
		return false
	}
	if err := writeClipboardText(code); err != nil {
		g.setInviteCopyFeedback("COPY FAILED: " + strings.ToUpper(err.Error()))
		networkDiagnostic("HOST invite code copy failed err=%v", err)
		return false
	}
	g.setInviteCopyFeedback("COPIED TO CLIPBOARD!")
	networkDiagnostic("HOST invite code copied to clipboard len=%d", len(code))
	return true
}

// hostJoinCodeLabelWithCopyHint is the host code label plus copy affordance or
// recent copy feedback.
func (g *Game) hostJoinCodeLabelWithCopyHint() string {
	label := g.currentHostDisplayJoinCodeLabel()
	if g.inviteCopyFeedback != "" && time.Now().Before(g.inviteCopyFeedbackUntil) {
		return label + "   >> " + g.inviteCopyFeedback
	}
	if code, ready := g.currentHostDisplayJoinCode(); ready && code != "" {
		return label + "   [CTRL+C / CLICK: COPY]"
	}
	return label
}

// ---------------------------------------------------------------------------
// Asynchronous join

type netJoinResult struct {
	session    *netplaySession
	visibility int
	scope      int
	err        error
}

type pendingNetJoin struct {
	target    string
	started   time.Time
	done      chan netJoinResult
	onSuccess func()
}

// startNetClientFromInput dispatches a join field value to the right client
// path. It only touches g.netplay and the online session visibility/scope, so
// it can run on a scratch Game from a background goroutine.
func (g *Game) startNetClientFromInput(address string) error {
	address = strings.TrimSpace(address)
	upper := strings.ToUpper(address)
	switch {
	case strings.HasPrefix(upper, cloudflareInviteCodePrefix):
		return g.StartNetClientCloudflareInvite(address)
	case strings.HasPrefix(upper, inviteCodePrefix):
		return g.StartNetClientInvite(address)
	case strings.HasPrefix(upper, shortOpenCodePrefix):
		return g.StartNetClientShortOpenCode(address)
	case strings.Contains(strings.ToLower(address), cloudflareTunnelHostSuffix):
		return g.StartNetClientServer(multiplayerServerInfo{
			Name: "CLOUDFLARE URL", Address: address, TunnelAddress: address, Transport: "cloudflare",
			Sources: "JOIN FIELD", Players: -1, MaxPlayers: maxOnlinePlayers, Protocol: netplayProtocolVersion,
		})
	default:
		return g.StartNetClient(address)
	}
}

// beginAsyncNetJoin runs connect on a background goroutine so the window keeps
// responding during dial + handshake (up to ~16 s). The result is installed on
// the game thread by updatePendingNetJoin.
func (g *Game) beginAsyncNetJoin(target string, connect func(scratch *Game) error, onSuccess func()) {
	if g == nil || connect == nil {
		return
	}
	_ = g.CloseNetplay()
	job := &pendingNetJoin{
		target:    target,
		started:   time.Now(),
		done:      make(chan netJoinResult, 1),
		onSuccess: onSuccess,
	}
	visibility, scope := g.onlineSessionVisibility, g.onlineSessionScope
	g.pendingNetJoin = job
	g.multiplayerFocus = false
	networkDiagnostic("JOIN async start target=%s", target)
	go func() {
		scratch := &Game{onlineSessionVisibility: visibility, onlineSessionScope: scope}
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("join crashed: %v", r)
				}
			}()
			err = connect(scratch)
		}()
		if err != nil && scratch.netplay != nil {
			_ = scratch.netplay.Close()
			scratch.netplay = nil
		}
		job.done <- netJoinResult{
			session:    scratch.netplay,
			visibility: scratch.onlineSessionVisibility,
			scope:      scratch.onlineSessionScope,
			err:        err,
		}
	}()
	g.refreshPendingNetJoinMessage()
}

func (g *Game) refreshPendingNetJoinMessage() {
	job := g.pendingNetJoin
	if job == nil {
		return
	}
	target := job.target
	if len(target) > 48 {
		target = target[:45] + "..."
	}
	seconds := int(time.Since(job.started).Seconds())
	g.multiplayerMessage = fmt.Sprintf("CONNECTING TO %s... %ds (ESC TO CANCEL)", strings.ToUpper(target), seconds)
}

// cancelPendingNetJoin abandons an in-flight join. The background goroutine
// cannot be interrupted mid-dial, so a drainer closes any session it produces.
func (g *Game) cancelPendingNetJoin() {
	job := g.pendingNetJoin
	if job == nil {
		return
	}
	g.pendingNetJoin = nil
	go func() {
		if result := <-job.done; result.session != nil {
			_ = result.session.Close()
		}
	}()
	networkDiagnostic("JOIN async cancelled target=%s after=%v", job.target, time.Since(job.started).Round(time.Millisecond))
}

// updatePendingNetJoin returns true while a join is in progress; the caller
// then skips normal menu input for this frame.
func (g *Game) updatePendingNetJoin() bool {
	job := g.pendingNetJoin
	if job == nil {
		return false
	}
	select {
	case result := <-job.done:
		g.pendingNetJoin = nil
		if result.err != nil {
			g.reportNetError("JOIN ERROR", result.err)
			return true
		}
		if result.session == nil {
			g.reportNetError("JOIN ERROR", fmt.Errorf("connection finished without a session"))
			return true
		}
		g.netplay = result.session
		g.onlineSessionVisibility = result.visibility
		g.onlineSessionScope = result.scope
		networkDiagnostic("JOIN async ready target=%s player=%d after=%v", job.target, result.session.assignedPlayer(), time.Since(job.started).Round(time.Millisecond))
		if job.onSuccess != nil {
			job.onSuccess()
		}
		return true
	default:
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.cancelPendingNetJoin()
		g.multiplayerMessage = "CONNECTION CANCELLED"
		return true
	}
	g.refreshPendingNetJoinMessage()
	return true
}

// reportNetError shows the complete error (rendered wrapped) and writes it to
// network.log.
func (g *Game) reportNetError(prefix string, err error) {
	if g == nil || err == nil {
		return
	}
	g.multiplayerMessage = prefix + ": " + err.Error()
	networkDiagnostic("%s: %v", prefix, err)
}

// ---------------------------------------------------------------------------
// Multi-line text

// wrapTextToWidth splits text into lines no wider than maxWidth according to
// measure. Words longer than a line (URLs, codes) are broken by character.
func wrapTextToWidth(text string, maxWidth float64, measure func(string) float64) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxWidth <= 0 || measure == nil {
		return []string{text}
	}
	var lines []string
	current := ""
	flush := func() {
		if current != "" {
			lines = append(lines, current)
			current = ""
		}
	}
	for _, word := range strings.Fields(text) {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if measure(candidate) <= maxWidth {
			current = candidate
			continue
		}
		flush()
		for measure(word) > maxWidth {
			runes := []rune(word)
			cut := 1
			for cut < len(runes) && measure(string(runes[:cut+1])) <= maxWidth {
				cut++
			}
			lines = append(lines, string(runes[:cut]))
			word = string(runes[cut:])
		}
		current = word
	}
	flush()
	return lines
}

func sourceMenuTextWidth(kind sourceMenuFontKind, size float64) func(string) float64 {
	face, err := sourceMenuFace(kind, size)
	if err != nil || face == nil {
		// Rough fallback so wrapping still happens without the font.
		return func(s string) float64 { return float64(len([]rune(s))) * size * 0.55 }
	}
	return func(s string) float64 {
		return float64(font.MeasureString(face, s).Ceil())
	}
}

// drawWrappedMenuText draws text wrapped into at most maxLines lines starting
// at (x, topY). The last line ends with "..." when text was cut; the full text
// is in network.log. It returns the number of lines drawn.
func drawWrappedMenuText(screen *ebiten.Image, text string, kind sourceMenuFontKind, size float64, clr color.Color, x, topY, maxWidth, lineHeight float64, maxLines int, background color.Color) int {
	measure := sourceMenuTextWidth(kind, size)
	lines := wrapTextToWidth(text, maxWidth, measure)
	if len(lines) == 0 {
		return 0
	}
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		last := []rune(lines[maxLines-1])
		for len(last) > 0 && measure(string(last)+"...") > maxWidth {
			last = last[:len(last)-1]
		}
		lines[maxLines-1] = string(last) + "..."
	}
	if background != nil {
		ebitenutil.DrawRect(screen, x-6, topY-3, maxWidth+12, float64(len(lines))*lineHeight+6, background)
	}
	for i, line := range lines {
		drawSourceMenuText(screen, line, kind, size, clr, x, topY+float64(i)*lineHeight)
	}
	return len(lines)
}

// wrappedLineCount reports how many lines text needs at the given width.
func wrappedLineCount(text string, kind sourceMenuFontKind, size, maxWidth float64) int {
	return len(wrapTextToWidth(text, maxWidth, sourceMenuTextWidth(kind, size)))
}
