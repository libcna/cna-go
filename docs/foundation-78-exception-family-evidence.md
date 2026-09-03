# Foundation 78 — the eight XNA exception types

```text
COMPLETE_TYPES   163 -> 171        MISSING_TYPE       94 -> 86
PARTIAL_TYPES      0 ->   0        MISSING_MEMBER      0 ->  0
BCL_INHERITED_PUBLIC_MEMBERS   24 -> 91
BLOCKED_DECLARED_MEMBERS        0 ->  2
```

Foundation 76 projected `System.Exception` as a signature adapter. This
milestone composes it as a **base** and projects every type that derives from it.

## Where the adapter lives, and why

The settled composition rule keeps a BCL base adapter unexported, so nothing
outside the module can reach the base state. That works while every consumer of
a base is in one package — `Collection<T>` has one consumer and
`Dictionary<K,V>` has one, both in the framework package.

`System.Exception` has **eight**, in **four** packages: Audio, Content, Graphics
and Storage. An unexported framework type is unreachable from any of them, so
the rule's own escape applies — *"when a COMPOSED base gains a consumer outside
the framework package the adapter moves to an internal package rather than
becoming exported"*. `internal/bclexception` is that package, and `internal/`
keeps the adapter unreachable from outside the module, which is the property the
unexported field had.

`System.Runtime.InteropServices.ExternalException` is composed over the **same**
adapter. It extends `System.SystemException`, which adds no public surface, so
its inherited set is `System.Exception`'s plus one — a public `ErrorCode` over
the protected `HResult` — and a `ToString` **override** the adapter selects with
a flag the derived constructor sets.

## A language accessor, and why it is not an escape hatch

Go has no explicit interface implementation, and no way for one package to
satisfy another package's *unexported* method. A reference interface whose
implementors live in four other packages therefore cannot hide its
distinguishing accessor, so `ExceptionReference` exposes `State()`.

That is an exported member the pinned CLR type does not declare, and it would
otherwise be an `UNEXPECTED_MEMBER`. It is admitted by a new registry field —
`LanguageAccessors` on the base adapter — which admits it **by name**, only on
the adapter's own Go type and on types that actually compose that base, and
which requires a reason. A member of the same name anywhere else is still
unexpected.

The accessor is what keeps the interface closed: `State()` returns
`*bclexception.State`, a type declared in an internal package, so no consumer
can name it and no consumer can invent a ninth exception type.

## Two declared members that are blocked, and counted

`ContentLoadException` and `StorageDeviceNotConnectedException` each declare a
**protected** `.ctor(SerializationInfo, StreamingContext)` of their own. That is
the same closure Foundation 74 measured at 238 public BCL members across
`System.Decimal`, `System.DateTime` and `System.Runtime.Serialization` — types
the profile names nowhere else. Neither constructor is reachable for a second
reason: the only caller of a deserialization constructor is a CLR formatter, and
CNA-Go deserialises nothing.

They are recorded in `blockedDeclaredMembers`, the declared-member counterpart
of a base adapter's `BCL_PROJECTION_BLOCKED_EXTERNAL` exclusion. It carries the
same weight — an admission, not a permission — and three things keep it measured:

- the identity must name a member the pinned contract really declares;
- the entry must state its kind, what it needs and why;
- the count is reported as **`BLOCKED_DECLARED_MEMBERS`**, separately from
  `MISSING_MEMBER`, so it can never be mistaken for surface that is present.

The pinned `3243` XNA-declared projections is now checked as
`declared + blocked == 3243`, so a blocked member shows up as a smaller
projection plus an equal, separately counted admission — never as a total that
quietly stayed the same.

## The derived class name is the load-bearing detail

`Exception::get_Message` renders

```text
Exception of type '{0}' was thrown.
```

from `GetClassName()`, and for a derived type that is the **derived** name. A
composed base cannot see its deriver, so the constructor supplies it, and every
one of the eight is pinned:
`NewDeviceLostExceptionByNone().Message()` names
`Microsoft.Xna.Framework.Graphics.DeviceLostException`, not `System.Exception`.

The three ExternalException subclasses differ again: their parameterless
constructor supplies the BCL's own `Arg_ExternalException` message rather than
leaving the field null, so they never render the default sentence at all.

## Three observable differences between the two ToStrings

| | `System.Exception` | `ExternalException` |
| --- | --- | --- |
| HResult | absent | ` (0x80004005)`, eight uppercase hex digits |
| inner exception | ` ---> ` + inner, then CRLF, three spaces and the end-of-stack marker | ` ---> ` + inner, and **no marker** |
| message separator | `": "` after a length test | `": "` after `IsNullOrEmpty` |

All three are pinned, in the packages that can see both halves.

## Why the interface widens at returns

`InnerException` returns `System.Exception`, and the CLR value is the **derived**
object. Returning a concrete `*Exception` would erase which of the nine kinds a
consumer chained — permanently. So the interface is the spelling at every
position, and the downcast a C# consumer writes as `catch (DeviceLostException)`
is `inner.(*graphics.DeviceLostException)`, which the external-consumer gate
performs across three packages.

## Falsifiability

Nineteen new tests across five packages, none skipped. The ones that would fail
a plausible wrong implementation:

- a base that named `System.Exception` instead of the deriver passes every other
  assertion and fails the default-message ones;
- an `ExternalException` subclass that inherited the base `ToString` renders no
  HResult and appends the marker, and both are asserted;
- an `E_FAIL` constant off by one hex digit — which this milestone actually got
  wrong on the first pass, and which the pinned `0x80004005` rendering caught.

## Qualification

```text
go test ./...                        PASS
go run ./tools/behavior              OBSERVATIONS=737 ASSERTIONS=737 FAILURES=0
go run ./tools/api_compat            TOTAL_DIAGNOSTICS=86, all MISSING_TYPE;
                                     MISSING_MEMBER=0, PARTIAL_TYPES=0,
                                     BLOCKED_DECLARED_MEMBERS=2
go run ./tools/native_abi ...        BOUND_FUNCTIONS=230 ABI_MISMATCHES=0
go run ./tools/external_consumer     TESTS=103 FAILURES=0 STATUS=PASS
go run ./tools/resource_strings      CLAIMED=49 VERIFIED=49 FINDINGS=0
```

The ABI is untouched: the whole family is managed.
