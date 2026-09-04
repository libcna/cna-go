# Foundation 91 — Storage, and the blocker that turned out to be half wrong

Two types, `StorageDevice` and `StorageContainer`, close
`Microsoft.Xna.Framework.Storage`. The exception,
`StorageDeviceNotConnectedException`, shipped with the exception family in
Foundation 78.

## Authority

| what | file | sha256 |
| --- | --- | --- |
| both types | `Microsoft.Xna.Framework.Storage.dll` | `798f678e9ae3d9af…` |
| the three System.IO enums, IAsyncResult | `mscorlib.dll` 4.0.30319.1 | `5634668d4775b011…` |
| `CannotEndTwice`, `InvalidStoragePath` | `Microsoft.Xna.Framework.dll` | `38e7093f52d7474b…` |
| native evidence | `~/deps/cna-c-abi-0.21.0/libcna_c_api.so` | `c32bfbd307d69566…` |

## Why this family and not content plumbing

The plan said content plumbing next. Measuring it corrected that:
`cna_content_reader_create` takes a `CNA_StorageStreamHandle`, and the only
routes in the whole ABI that produce one are in `storage.h`. The real chain is

```
Storage  ->  content plumbing  ->  the Model family's native draw slice
```

Three pure-managed content types were written before this was measured and are
parked, not committed — they compile only alongside `ContentReader`, which
cannot be created until this family exists.

## Half the recorded blocker was wrong, and the measurement says so

`System.IO.BinaryReader` has been DEFERRED as a base since Foundation 29 with
this detail: *"CNA-Go maps System.IO.Stream to io.Reader, which carries neither
seeking nor the encoding-aware Read7BitEncodedInt behavior the reader's
inherited surface depends on."*

The seeking half does not survive measurement:

```
actual `Stream::Seek(` calls in ContentReader's whole IL:   0
`get_CanSeek` guards:                                        1
```

There is no `Seek` anywhere — an earlier count of "1" was the substring of
`get_CanSeek`. That single guard sits in `private static Stream PrepareStream`,
which is neither inherited nor contract surface, and the size check behind it is
skipped entirely when the stream cannot seek. **The reference already supports a
non-seekable stream.** `Read7BitEncodedInt` is the half that is real.

That correction belongs to the content family, but it was found here and is
recorded here.

## System.IO.Stream at a read-WRITE return position

Foundation 53 built `writtenStreamParameters` to rewrite the `io.Reader`
default at PARAMETER positions the reference writes — `SaveAsPng`, `SaveAsJpeg`
— with the note that *"direction cannot be derived from the signature"*, since
both directions are spelled `Stream`.

Its note also said "every stream position in the profile is read". That was true
of RETURN positions when written. `StorageContainer` is where it stops: a save
file is opened to be written, and `CreateFile` exists for nothing else.

So the mechanism gained its return-position counterpart,
`readWriteStreamReturns`, naming four members. The Go type is
`io.ReadWriteSeeker` rather than `io.ReadWriter` because the CNA stream behind
it carries seek, length and position, and a save file that could not seek would
not be one. Seventeen Stream positions exist in the contract; four are these.

## XNA's storage APM is FAKE async, measured from XNA

`BeginShowSelector`'s whole body after its two guards:

```
var result = new StorageDeviceAsyncResult(state, player);
if (callback != null) callback.Invoke(result);
return result;
```

The callback fires **before** `Begin` returns, from managed code. There is no
thread, no queue, and nothing to wait for. So `IsCompleted` and
`CompletedSynchronously` are always true and the wait handle is always
signalled — those are not simplifications of a general case, they are what this
async result IS.

CNA's own header says the same of its side — *"XNA's fake-async
BeginXxx/EndXxx pair, which CNA completes synchronously"* — but the claim above
rests on the reference. CNA agreeing is a convenience, not the evidence.

`System.Threading.WaitHandle` projects to a receive-only channel that is always
already closed, which is the Go type whose ROLE it is.

### The two failures End has

```
var typed = result as StorageDeviceAsyncResult;
if (typed == null) throw new ArgumentNullException("result");   // for a WRONG TYPE too
if (typed.endHasBeenCalled)
    throw new InvalidOperationException(FrameworkResources.CannotEndTwice);
typed.endHasBeenCalled = true;
```

A result of the wrong TYPE raises `ArgumentNullException`, not
`ArgumentException` — the `isinst` and the null test collapse into one branch.
The `endHasBeenCalled` latch is projected as REMOVING the pairing from a
registry: a second End finds nothing and answers the same refusal.

## The reference's own containment guard

Nine of `StorageContainer`'s members reach the filesystem and every one opens
with `ValidateArguments`:

```
VerifyNotDisposed();
if (String.IsNullOrEmpty(path)) throw new ArgumentNullException(argumentName);
string full = GetFullPath(path);
if (!full.StartsWith(_rootPath))
    throw new ArgumentException(FrameworkResources.InvalidStoragePath, argumentName);
```

A path that resolves outside the container's root is refused **by the
reference**. So "a game cannot reach the user's documents through this type" is
not a rule this projection adds; it is what XNA does, and reproducing it is the
job. Three details in order: disposal first, then the empty-path refusal — which
is `ArgumentNullException`, not `ArgumentException` — then the root check.

## Where the stress slice is allowed to write, proved rather than assumed

The standing constraint is a project-controlled root and no host directories.
The slice did not assume one. It printed the root first, and the root was

```
/home/robertvokac/.local/share/cna-go-stress-storage
```

— outside the project, in the user's home. CNA builds it from `XDG_DATA_HOME`,
so the harness redirects that into the repository; but the slice does not trust
the redirect either. It reads the root back and **refuses to continue** unless
the root lies under the permitted path:

```go
if !strings.HasPrefix(root, permitted) {
    return fmt.Errorf("storage root %q is outside the permitted root %q; refusing to touch it", ...)
}
```

A run that cannot prove containment does nothing and says so. That is the same
shape `MICROPHONE_CAPTURE_CALLS` has: a safety claim is worth more when
something measures it.

`STORAGE_ROOT_CHECKS` is the counter that proves it ran.

## Four verifier gaps this family exposed, all fixed generally

None is a special case for Storage.

- **Package qualification for BCL types this module declares.** `TimeSpan` was
  the only one and had a hand-written `if`. Five entries now need it, so the
  condition became a registry, `frameworkDeclaredBCLTypes`.
- **The qualifier goes AFTER the pointer.** A CLR interface projects to a Go
  pointer, so `*framework.AsyncResult` and not `framework.*AsyncResult`.
- **`System.AsyncCallback` was degrading to `any`.** It is a delegate, and the
  settled rule for a delegate is a Go func — the same rule `EventHandler<T>`
  gets. Degrading it would erase the argument an APM callback exists to receive.
- **Static-event accessor detection.** The check found a removal by
  `HasPrefix("Remove")`, which a STATIC event's type-prefixed name does not
  satisfy: `StorageDeviceRemoveDeviceChangedHandler` does not start with
  "Remove". Every projected event before this one was an instance event.

## Two members that look like field reads and are not

- `get_StorageDevice` is fallible: the reference reads the field AFTER
  `VerifyNotDisposed`, as every property on the type does.
- `get_IsDisposed` is the one property that does NOT verify — asking a disposed
  object whether it is disposed has to work — and is fallible for a different
  reason: it reaches CNA.

And `Finalize` is projected as what the finalizer does: a release that does NOT
raise `Disposing`, because the reference's `Dispose(false)` skips the managed
half.

## Falsifiability

Fifty-one defects were planted: thirty-nine managed, eleven native, and one
containment guard that needed a table of its own.

### Two survivors that were harness faults, not test gaps

`selector_widens_the_player_range` survived because its anchor matched **prose**.
The doc comment quotes the reference:

```
//	if (player != 0xff && (player < 0 || player > 3))
```

and the real guard is twenty-one lines below it. `replace(..., 1)` mutated the
comment and left the projection untouched. This is exactly the trap Foundation
86 recorded, walked into again — the anchor now carries a tab and the Go
spelling the prose cannot have.

`selector_player_sentinel_is_wrong` survived because the TEST read the same
constant the code did: it passed `framework.PlayerIndex(storageNoPlayer)`, so
changing the constant moved the expectation with it. The literal is written out
now, with a separate assertion that the sentinel IS 255. That is the value half
of the same rule: an anchor must not match prose, and an expectation must not be
computed from the thing it checks.

### Two survivors that belonged in the other table

`no_player_overload_names_player_zero` and
`player_overload_forwards_the_wrong_directory_count` change no guard, so no
managed test could ever see them. They were moved to the native table, where
they still survive — see below.

### The containment guard needed a NEGATIVE CONTROL

`storage_slice_skips_the_containment_proof` cannot be killed by an honest run:
the root really is inside the permitted directory, so removing the check changes
nothing. A safety guard whose failure path is never exercised is a comment.

So it has its own table, run with `CNA_GO_STORAGE_ROOT` pointed somewhere the
storage root cannot possibly be under. And the signal is **not the exit code** —
with the guard removed the slice happily works against a root it was never
permitted to touch, and that run SUCCEEDS, which is precisely the danger. What
proves the guard is that a refused run does no storage work at all, so the
control asserts `STORAGE_SELECTOR_CYCLES` and `STORAGE_FILE_WRITES` are zero.

### Three native survivors that were slice gaps

- `create_file_opens_instead` survived on a leftover: a run that fails part-way
  leaves `stress.dat` behind, and the next run cannot then tell `CreateFile`
  from an `OpenFile` of something that already existed. The slice now starts
  from a known state.
- `stream_read_never_reports_eof` survived because the slice read with
  `io.ReadFull` into an exact-size buffer, which never needs EOF. It now reads
  PAST the end, which is the only place that translation shows.
- `delete_file_does_nothing` survived because nothing checked the file was gone.

### Totals

| table | planted | killed | survived |
| --- | ---: | ---: | ---: |
| managed | 39 | 39 | 0 |
| native | 11 | 8 | 3 |
| containment | 1 | 1 | 0 |

### The three unkilled survivors, named

1. `no_player_overload_names_player_zero` and
2. `player_overload_forwards_the_wrong_directory_count` — both change WHICH CNA
   selector route is called. On a host with a single storage location every one
   of the four routes succeeds, so the choice is not observable through the
   projection's own surface. A machine with two storage locations would
   distinguish them.
3. `dispose_does_not_reach_cna` — `IsDisposed` answers the managed latch first,
   so CNA is never asked and a Dispose that skipped it looks identical. That is
   a consequence of the design rather than a gap in it: the managed latch IS the
   answer, and the reference's own `IsDisposed` reads a managed field too.

## Scoreboard

| counter | before | after |
| --- | ---: | ---: |
| COMPLETE_TYPES | 206 | 208 |
| MISSING_TYPE | 51 | 49 |
| PARTIAL_TYPES | 0 | 0 |
| MISSING_MEMBER | 0 | 0 |
| GLOBAL_UNREVIEWED | 0 | 0 |
| BOUND_FUNCTIONS | 350 | 393 |
| FRONTIER_FAMILIES | 7 | 6 |
| ABI_MISMATCHES | 0 | 0 |

Forty-three routes bound. Six were left unbound for having no call site — two
subscribe pairs and the container's type name. Three `_ext` routes ARE bound and
are not XNA surface: they are how the harness isolates itself and proves it.
