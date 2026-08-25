# Foundation 25 — internal XNA interfaces, and the frontier that remains

Foundation 25 closes the last measurement hole the interface work left open and
then re-derives the whole frontier. It completes no XNA type: after it, every
remaining candidate is blocked on something this session's decisions do not
resolve, and the evidence for each blocker is recorded rather than assumed.

## The hole

Foundation 24 made every **non-XNA** direct interface a declared relationship.
It left the mirror case open: an XNA-namespaced interface that is not a public
contract type. `buildMappedInterfacesAndWitnesses` still dropped those with a
bare `continue`, and the frontier still counted them as unmapped names.

Two exist, both in `Microsoft.Xna.Framework.Graphics.dll`:

```text
.class interface private abstract auto ansi beforefieldinit
       Microsoft.Xna.Framework.Graphics.IGraphicsResource
    ReleaseNativeObject(bool) / SaveDataForRecreation() / RecreateAndPopulateObject()

.class interface private abstract auto ansi beforefieldinit
       Microsoft.Xna.Framework.Graphics.IDynamicGraphicsResource
    ContentLost / IsContentLost / SetContentLost(bool)
```

`private` at class scope means assembly-visible, which is why both are
correctly absent from the 257-type public contract while seven and four public
graphics types respectively declare them.

They contribute no public surface for a **stronger** reason than a BCL interface
does: they have no public member to project at all. Both are now declared
`INTERNAL_NO_SURFACE`, and an XNA interface that is neither a public contract
type nor a declared internal one is `INTERFACE_MAPPING_MISMATCH`.

The full relationship table now covers ten interfaces, every row measured with
`projectedMembers == 0`:

```text
INTERNAL_NO_SURFACE  IGraphicsResource                          3 members,  7 declaring types
INTERNAL_NO_SURFACE  IDynamicGraphicsResource                   3 members,  4 declaring types
MAPPED_NO_SURFACE    System.IEquatable`1                        1 member,  40
MAPPED_NO_SURFACE    System.IDisposable                         1 member,  29
MAPPED_NO_SURFACE    System.Collections.Generic.IEnumerable`1   1 member,  12
MAPPED_NO_SURFACE    System.Collections.Generic.IEnumerator`1   4 members,  5
MAPPED_NO_SURFACE    System.Collections.Generic.ICollection`1   7 members,  1
MAPPED_NO_SURFACE    System.Collections.Generic.IList`1         4 members,  1
MAPPED_NO_SURFACE    System.IComparable`1                       1 member,   1
MAPPED_NO_SURFACE    System.IServiceProvider                    1 member,   1
```

## The regenerated frontier

With `EventArgs`, `EventHandler<T>`, `IDisposable`, the already-settled BCL
interfaces, and the two internal ones all accounted for, fourteen missing types
are dependency-complete. **All fourteen are blocked on behavior**, not on a
mapping:

```text
capability / device state
    Graphics.DisplayMode                assembly ctor; display enumeration
    Input.GamePadCapabilities           assembly/private ctor; device capability
    Input.Touch.TouchPanelCapabilities  no constructor at all; device capability
    Input.Mouse                         device state plus MouseMessageHooker
    Audio.RendererDetail                assembly ctor; XACT renderer enumeration
    Media.MediaSource                   assembly ctor; media backend
    GameWindow                          abstract, assembly ctor, 8 bodyless
                                        public abstract members, real HWND

runtime not present
    Audio.AudioCategory                 assembly ctor needing AudioEngine; every method P/Invokes
    Audio.Cue                           XACT native
    Audio.SoundEffectInstance           assembly ctors; 18 throw sites through the native error code
    Graphics.EffectAnnotation           assembly ctor; unmanaged calli
    Media.Video                         assembly ctor needing GraphicsDevice and a content file
    FrameworkDispatcher                 drives the microphone, dynamic-audio, media and storage pumps
    TitleContainer                      TitleLocation.Path and the file system
```

`GameWindow` is the only one that was not already on the deferral list, and it
was re-derived this session rather than assumed. See
`foundation-24-idisposable-relationship-evidence.md`.

## What a mapping would still unlock

Six BCL shapes remain, and for each the exact set of types whose **only**
remaining blocker it is:

```text
5  System.Attribute                     the five ContentSerializer* attributes
4  System.Exception                     NoMicrophoneConnectedException, DeviceLostException,
                                        DeviceNotResetException, NoSuitableGraphicsDeviceException
2  ReadOnlyCollection`1                 Audio.Microphone, Media.VisualizationData
1  System.Collections.ObjectModel.Collection`1   GameComponentCollection
1  System.Collections.Generic.Dictionary`2       LaunchParameters
1  System.Action`1                      Content.ContentManager
```

`Microphone` and `ContentManager` stay device- and filesystem-blocked even with
their shape mapped, so the mappings that would genuinely add a usable type are
the collection bases and the exception hierarchy.

## GameComponentCollection — derived, and blocked on a real decision

The prompt-level guidance said to reconsider it once `IGameComponent`, the event
projection, and `GameComponentCollectionEventArgs` were complete. All three now
are, so it was re-derived from
`Microsoft.Xna.Framework.Game.dll` in full. Its behavior **is** entirely
managed and IL-provable:

```text
InsertItem(index, item)   IndexOf(item) != -1 -> throw ArgumentException
                          ("cannot add the same component multiple times");
                          base.InsertItem; then, only if item is non-null,
                          raise ComponentAdded with a FRESH
                          GameComponentCollectionEventArgs(item)
RemoveItem(index)         read base[index] first; base.RemoveItem;
                          then, only if the removed item was non-null,
                          raise ComponentRemoved
SetItem(index, item)      unconditionally throws NotSupportedException;
                          the indexer setter is not supported at all
ClearItems()              raise ComponentRemoved for EVERY item, index 0
                          upward, and only THEN call base.ClearItems()
```

That last line is the non-obvious part and the reason the type must not be
guessed at: `Insert` and `Remove` mutate **before** they raise, while `Clear`
raises for the whole collection **before** it mutates. `OnComponentAdded` and
`OnComponentRemoved` are `private`, null-check the delegate, and pass the
collection itself as sender.

It is nevertheless **not implementable today**, for a reason that is about
projection rather than behavior. Its declared public/protected surface is only
seven members:

```text
.ctor()
InsertItem, RemoveItem, SetItem, ClearItems   (all `family`, i.e. protected overrides)
ComponentAdded, ComponentRemoved              (events)
```

Everything a caller actually uses — `Add`, `Remove`, `Clear`, `Count`, the
indexer, `IndexOf`, `Insert`, `RemoveAt`, `GetEnumerator` — is **inherited from
`Collection<IGameComponent>`** and is not a declared member of the XNA type.
Projecting only the seven declared members would turn the strict gate green
while producing a collection **nothing can be added to**, and the four protected
overrides would have no base to override. That is a scoreboard improvement with
no capability behind it, which the project's rules exist to prevent.

`LaunchParameters` is the same decision in starker form: it derives from
`Dictionary<string,string>` and its **only** declared member is a public
parameterless constructor.

So both are blocked on one material public-API decision that neither this
session's prompt nor existing repository policy resolves: **how a BCL collection
base projects into a Go type that cannot inherit.** It governs
`Collection<T>`, `ReadOnlyCollection<T>`, and `Dictionary<K,V>`, and therefore
`GameComponentCollection`, `LaunchParameters`, `Microphone`,
`VisualizationData`, and the four `Model*` types.

The alternatives, recorded rather than chosen:

1. **Re-declare the inherited surface on the derived Go type.** Faithful for
   callers, but those members are not declared members of the XNA type, so it
   needs a new expected-member category for inherited BCL surface, a rule for
   which inherited members are in scope, and new count arithmetic in the
   `expectedCountFormula`.
2. **Project only the declared members.** Faithful to the contract's letter,
   useless in practice, and leaves four protected overrides overriding nothing.
3. **Make the BCL collection base a language adapter** — a framework-package
   `Collection[T]` the derived type holds and re-exports. Still requires
   decision 1's answer about which surface is projected, and adds a public
   support type that is not XNA.
4. **Defer**, which is what this milestone does.

## Scoreboard — unchanged, deliberately

```text                        before    after
TARGET_TYPES                   118      118
TARGET_MEMBERS                1722     1722
TOTAL_DIAGNOSTICS              316      316
MISSING_TYPE                   139      139
MISSING_MEMBER                 177      177
COMPLETE_TYPES                 113      113
PARTIAL_TYPES                    5        5

mutation inventory             446      447
behavior corpus                575      575
```

Every mismatch, leak, allowlist, and unmeasured counter remains zero. CNA ABI is
unchanged and nothing was rebuilt.
