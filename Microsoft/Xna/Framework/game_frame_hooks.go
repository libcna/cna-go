package framework

// This file projects Game's four remaining frame-boundary protected virtuals:
//
//	.method family hidebysig newslot virtual instance void BeginRun()
//	.method family hidebysig newslot virtual instance void EndRun()
//	.method family hidebysig newslot virtual instance bool BeginDraw()
//	.method family hidebysig newslot virtual instance void EndDraw()
//
// They are ordinary projected members here, and not GameCallbacks members, for
// the same reason GameComponent::OnEnabledChanged is an ordinary method: the
// mapper redirects exactly the five protected virtuals GameCallbacks declares
// -- Initialize, LoadContent, Update, Draw, UnloadContent -- and nothing else.
// A protected virtual outside that set projects as a method on the declaring
// type whose body is the reference's base body.
//
// # The two run hooks are authoritative no-ops
//
//	BeginRun()   IL_0000: ret      // code size 1
//	EndRun()     IL_0000: ret      // code size 1
//
// Both are declared so a derived class has something to override. Neither does
// anything, and neither of these does anything either. This is the same
// provenance GameBaseLoadContent and GameBaseUnloadContent already carry.
//
// # The two draw hooks read one private field, and CNA-Go never assigns it
//
//	BeginDraw()                                          // code size 36
//	  if (graphicsDeviceManager != null &&
//	      !graphicsDeviceManager.BeginDraw()) return false;
//	  Logger.BeginLogEvent((LoggingEvent)4, "");
//	  return true;
//
//	EndDraw()                                            // code size 31
//	  if (graphicsDeviceManager != null) graphicsDeviceManager.EndDraw();
//	  Logger.EndLogEvent((LoggingEvent)4, "");
//
// Game::graphicsDeviceManager is a private field with exactly one assignment
// in the whole class, at the head of RunGame:
//
//	graphicsDeviceManager = Services.GetService(typeof(IGraphicsDeviceManager))
//	                        as IGraphicsDeviceManager;
//	if (graphicsDeviceManager != null) graphicsDeviceManager.CreateDevice();
//
// CNA-Go does not perform that resolution, and the reason is architectural
// rather than an oversight. The step immediately following it calls
// CreateDevice on whatever it found, and in CNA-Go the native runtime owns the
// device and creates it itself; performing the resolution without the
// CreateDevice that the reference pairs it with would produce a Game that had
// found a manager it never asked to create anything. Foundation 30 separately
// audited and recorded that nothing in CNA-Go can register into
// IGraphicsDeviceManager anyway: the projected GraphicsDeviceManager is partial
// and satisfies neither service contract, so the reference's only registrar
// cannot run.
//
// The field is therefore permanently null here, which is a state the reference
// itself has whenever no manager is registered -- an XNA game with no
// GraphicsDeviceManager reaches exactly these branches. Both bodies are
// reproduced as written; the null branch is simply the one that is always
// taken. Nothing is faked and no value channel is invented.
//
// Logger.BeginLogEvent and Logger.EndLogEvent are the same UNOBSERVABLE
// deferral GameBaseUpdate already records: Microsoft.Xna.Framework.Logger
// writes to an ETW-style sink that no projected member can read, so
// reproducing or omitting the call is indistinguishable from the public
// surface.
//
// # Nothing calls these automatically, and that is deliberate
//
// In the reference, RunGame calls BeginRun and EndRun and DrawFrame calls
// BeginDraw and EndDraw, all by virtual dispatch. CNA has canonical native
// hooks at those exact four positions -- CNA_GameFrameHooks::begin_run,
// end_run, begin_draw and end_draw -- and CNA-Go deliberately does not install
// them.
//
// Foundation 31 settled the rule this follows from: base behavior is never
// automatic. Installing the native hooks and forwarding them into these bodies
// would run the base at a position CNA-Go picked, and would make that base call
// mandatory. There is no override mechanism for these four members today, so
// the base is all there is and the forwarding would be provably inert -- but it
// would also prejudge the very decision an override mechanism has to make,
// which is whether and where the derived class calls the base.
//
// The hooks are audited, measured and left uninstalled. See
// docs/foundation-35-game-frame-hook-evidence.md for the measured native
// ordering and the exact public design choice that remains open.
//
// # Fallibility
//
// All four are fallible for the reason every non-stored Game member is: Game is
// a native-backed facade. The only failure any of them actually has is the
// Go-only guard that Game.Run, Game.Exit and the whole GameBase... family
// already report -- Go, unlike CLR, lets a consumer write &framework.Game{} and
// get an object whose managed state was never allocated. No reference body here
// has a throw site, so nothing else can fail.

// BeginRun is Game::BeginRun, a protected virtual whose body is one `ret`.
//
// The reference calls it from RunGame after Initialize returns and inRun is
// raised, and before the priming Update. Nothing calls it here; see the file
// comment for why the native begin_run hook is audited and left uninstalled.
func (g *Game) BeginRun() error {
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	return nil
}

// EndRun is Game::EndRun, also a bare `ret` of code size 1.
//
// The reference calls it from RunGame immediately after the blocking
// host.Run() returns, which is the same position CNA's end_run hook occupies.
func (g *Game) EndRun() error {
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	return nil
}

// BeginDraw is Game::BeginDraw, and its Boolean result is a real value channel
// rather than a success flag.
//
// DrawFrame is where it matters:
//
//	if (this.BeginDraw())
//	{
//	    ... set up gameTime ...
//	    this.Draw(gameTime);
//	    this.EndDraw();
//	    this.doneFirstDraw = true;
//	}
//
// A false answer skips Draw AND EndDraw for that frame. The same is true of
// CNA's canonical begin_draw hook, measured against the pinned runtime: a frame
// whose out_should_draw is set to CNA_FALSE delivers neither draw nor end_draw.
//
// The Boolean and the error are separate channels and are kept separate, which
// is the mapping this project already applies to IGraphicsDeviceManager's own
// BeginDraw. The Boolean says whether the frame draws; the error says whether
// the call could be made at all.
//
// The base body answers false only when a registered IGraphicsDeviceManager
// refuses. CNA-Go never assigns that private field -- see the file comment --
// so the base answers true, which is exactly what the reference answers for a
// Game with no manager registered.
func (g *Game) BeginDraw() (bool, error) {
	if !gameManagedState(g) {
		return false, errGameNotConstructed
	}
	// if (graphicsDeviceManager != null && !graphicsDeviceManager.BeginDraw())
	//     return false;
	// The field is null, so the guarded branch is not taken and the method
	// falls through to its unconditional `ldc.i4.1; ret`.
	return true, nil
}

// EndDraw is Game::EndDraw, the mirror of BeginDraw's manager step with no
// value channel of its own.
//
// The reference calls it from DrawFrame after Draw returns, and only on the
// frames BeginDraw admitted. Its whole body is the manager call and the logging
// event, so with no manager registered it is observably a no-op and this does
// nothing.
func (g *Game) EndDraw() error {
	if !gameManagedState(g) {
		return errGameNotConstructed
	}
	// if (graphicsDeviceManager != null) graphicsDeviceManager.EndDraw();
	return nil
}
