# Foundation 76 — `System.Exception`, and the profile's last missing member

```text
COMPLETE_TYPES   158 -> 159        MISSING_MEMBER      1 -> 0
PARTIAL_TYPES      1 ->   0        MISSING_TYPE       98 -> 98
TOTAL_DIAGNOSTICS 99 ->  98        every mismatch category  0
```

**`PARTIAL_TYPES` and `MISSING_MEMBER` are both zero.** Every type CNA-Go
projects, it now projects completely, and `TOTAL_DIAGNOSTICS` is exactly the
count of types it does not project at all.

## Reference authority, and a new one

```text
mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
  5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
Microsoft.Xna.Framework.Game.dll
  b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0
```

Every "read from the pinned mscorlib" claim in this repository has named that
sha256 since Foundation 26, but nothing could *check* a message read from it —
`tools/resource_strings` only knew about the retained XNA assemblies. The binary
with that exact hash is present on this machine, and it is now retained at
`~/deps/bcl-4.0-pinned/mscorlib.dll` and admitted by hash: the tool refuses it if
its content does not match.

That closed a real gap rather than a hypothetical one. Foundation 74 wrote two
`Dictionary<K,V>` messages from the IL's `ExceptionResource` names — the exact
mistake Foundation 49 existed to stop — and nothing checked them. Both turn out
to be right, and both are now verified byte for byte, along with the three this
milestone adds. The tool reports 49 claimed and 49 verified across 18 assemblies.

## Three roles for one CLR type

`System.Exception` needed all three at once, which no BCL type in the profile
had before:

| role | Go | why |
| --- | --- | --- |
| private base adapter | `exceptionBase` | the eight XNA exception types compose it |
| public concrete type | `Exception` | a consumer writes `new Exception("...")` |
| signature spelling | `ExceptionReference` | every projected position carries it |

### Why the signature position is an interface, at RETURNS too

The settled substitutable-base rule widens a base-typed **parameter** to an
exported reference interface and leaves a base-typed **return** as the concrete
pointer, recording the lost downcast as a language limitation.

This is the family where that trade would cost the type its purpose. An
exception hierarchy exists to be told apart by type; `InnerException` returning
a concrete `*Exception` would erase which of the eight kinds a consumer chained,
permanently. So `System.Exception` widens at **every** position, and the
downcast a C# consumer writes as `catch (DeviceLostException)` is the Go type
assertion `inner.(*graphics.DeviceLostException)`.

The interface carries an unexported accessor, so only this module can satisfy
it: a consumer cannot invent a ninth exception type and hand it to a projected
member.

### It is not a Go `error`

Foundation 29 recorded this as the material question and framed it as a
dilemma — either every settled per-operation fallibility decision reopens, or
the eight types are inert.

Neither branch is what this milestone takes. **The two are different
contracts.** A projected *operation* reports failure through a Go error, as it
has since Foundation 1, and not one of the roughly thirty per-operation
decisions is touched. A CLR exception *object* is a value the profile
constructs and passes — to `ShowMissingRequirementMessage`, and through
`InnerException` — and it needs a Go spelling or the members that carry it
cannot be projected at all. Giving the type an `Error() string` method would
collapse the two and publish a member the pinned CLR type does not declare, so
it has none.

The types are not inert either: `ShowMissingRequirementMessage` takes one today,
and the eight derived types will be constructed and chained.

## Eight of eleven members, and three measured blockers

| CLR member | outcome |
| --- | --- |
| `Message` | projected |
| `InnerException` | projected |
| `GetBaseException()` | projected |
| `StackTrace` | projected, and always empty — see below |
| `HelpLink` get/set | projected |
| `Source` get/set | projected |
| `ToString()` | projected |
| `GetType()` | projected |
| `Data` | `BCL_PROJECTION_BLOCKED_EXTERNAL` — `System.Collections.IDictionary` |
| `TargetSite` | `BCL_PROJECTION_BLOCKED_EXTERNAL` — `System.Reflection.MethodBase` |
| `GetObjectData` | `BCL_PROJECTION_BLOCKED_EXTERNAL` — the 238-member serialization closure Foundation 74 measured |

`StackTrace` is the member Foundation 29 called unrepresentable, and it stays
that — but the reason is sharper than "Go has no CLR stacks". The CLR captures
the frames **at throw time**, and CNA-Go throws no CLR exception, so
`_stackTraceString` is null for every state this projection can reach and the
reference's own getter would answer null. `System.String` projects to Go
`string`, which has no null, so it is the empty string. Returning a *Go* stack
would be a different thing wearing the same name.

`Message` is the opposite case, and it is why the projection carries a
`messageSet` flag the reference does not have. `new Exception()` genuinely
leaves `_message` null and renders

```text
Exception of type 'System.Exception' was thrown.
```

while `new Exception("")` renders the empty string. Go's `string` cannot tell
those apart on its own, and a consumer reading `Message` can.

## `Game.ShowMissingRequirementMessage`

```text
Game::ShowMissingRequirementMessage      host == null ? false
                                                      : host.ShowMissingRequirementMessage(e)
GameHost::ShowMissingRequirementMessage  ldc.i4.0; ret
WindowsGameHost::...
    e is NoSuitableGraphicsDeviceException -> MessageBox(NoSuitableGraphicsDevice + "\n\n" + e.Message)
    e is NoAudioHardwareException          -> MessageBox(NoAudioHardware)
    otherwise                              -> base, which is false
```

Both dialog branches are selected by an `isinst` against a specific exception
**type**, and neither of those two types is projected yet — so no consumer can
construct one, and no argument this member can be given reaches a dialog. The
reachable body is the base host's `false`, and that is what is projected: a
measured constant, not a chosen one. The member is infallible because the
projected body reaches nothing, which is one of the three shapes
`managedStoredMembers` admits.

CNA *does* offer `cna_message_box_show_simple_ext` and a test backend that
records instead of blocking. Binding it today would be a route with no
production call site, which the settled rule refuses; both branches become
reachable the milestone those two exception types are projected.

## Why the base is still deferred, and what it now waits on

`System.Exception` as a **base** stays `DEFERRED`, and the two roles are
independent exactly as `ReadOnlyCollection<T>`'s are. But the blocker has
changed from an open architecture question to a mechanical one:

- a `COMPOSED` base adapter must be **unexported** — the settled composition
  rule, enforced by `TestBCLBaseCompositionIsMeasuredNotAssumed` — and the eight
  derived exception types live in **four other packages**, so the adapter has to
  move to an internal package before any of them can hold it;
- `ContentLoadException` and `StorageDeviceNotConnectedException` additionally
  declare a **protected** `.ctor(SerializationInfo, StreamingContext)` of their
  own, which is the same serialization closure that blocks `GetObjectData`.

Both are recorded in `deferredDetail`, so the entry says what it waits on rather
than restating Foundation 29's dilemma, which this milestone answered.

## A verifier rule that was a fact about one adapter

`measureBCLSignatureAdapters` refused a signature adapter that declared a public
setter, with the message "which the read-only projection does not model". That
was a true statement about the one adapter that then existed —
`ReadOnlyCollection<T>`, a read-only view — dressed up as a rule about signature
adapters in general. `System.Exception` declares public `HelpLink` and `Source`
setters.

The check now walks the inventory per **accessor**: a settable property is two
Go identities under the settled member rule, the getter under the property's
name and a `Set`-prefixed pointer method. The read-only claim moved to where it
belongs — `readOnlyViewAdapters`, a closed list of the adapters whose CLR type
really is a view — and the four mutation fixtures that plant `Add`, `Clear`,
`SetItem` and `Items` on `ReadOnlyCollection` still reject every one, because
they are caught by the no-extra-member scan.

## Qualification

```text
go test ./...                        PASS
go run ./tools/behavior              OBSERVATIONS=737 ASSERTIONS=737 FAILURES=0
go run ./tools/api_compat            TOTAL_DIAGNOSTICS=98, ALL of them
                                     MISSING_TYPE; MISSING_MEMBER=0,
                                     PARTIAL_TYPES=0, every mismatch, leak,
                                     unexpected and allowlist category 0
go run ./tools/native_abi ...        BOUND_FUNCTIONS=230 ABI_MISMATCHES=0
go run ./tools/external_consumer     TESTS=101 FAILURES=0 STATUS=PASS
go run ./tools/resource_strings      CLAIMED=49 VERIFIED=49 FINDINGS=0
```

The ABI is untouched: this milestone binds no route.
