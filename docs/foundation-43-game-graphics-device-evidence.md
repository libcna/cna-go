# Foundation 43 — `Game.GraphicsDevice`, and a corrected resource string

Two things, both small and both about getting the reference exactly right.

## `Game.GraphicsDevice`

The last of `Game`'s members that is reachable without a missing type, a missing
assembly or a new lifecycle decision.

```csharp
IGraphicsDeviceService service = this.graphicsDeviceService;
if (service == null)
{
    service = this.Services.GetService(typeof(IGraphicsDeviceService))
              as IGraphicsDeviceService;
    if (service == null)
        throw new InvalidOperationException(Resources.NoGraphicsDeviceService);
}
return service.GraphicsDevice;
```

**The fallback is what makes it reachable.** `Game::graphicsDeviceService` has
exactly one assignment in the whole class, inside `HookDeviceEvents`, which base
`Initialize` calls and which CNA-Go records as a deferred step — so the cached
branch is never taken here.

That is not a gap being worked around. It is the state the reference itself is in
before `Initialize` runs, and the reference's own answer to it is the resolution
above. A consumer who registers an `IGraphicsDeviceService` into `Game.Services`
gets a device from this member exactly as they would in XNA; one who registers
nothing gets the reference's own `InvalidOperationException`.

It is projected as `graphics.GameGraphicsDevice(game)` rather than a method on
`Game`, by the settled cross-package cycle rule: the framework package cannot
name `GraphicsDevice`, because the Graphics package imports it.

**Nothing is faked.** CNA-Go publishes no `IGraphicsDeviceService` of its own —
`GraphicsDeviceManager` is a partial native-backed facade satisfying neither
service contract — so with no consumer-supplied service the member reports the
reference's failure rather than inventing a device. And when a service *is*
registered, the device it returns is the consumer's own, forwarded unchanged:
the reference's null check is on the **service**, not on the device, so a service
publishing none answers nil with no error.

```text                    before   after
TOTAL_DIAGNOSTICS           282     281
MISSING_MEMBER              147     146
Game missing members          9       8
```

## The corrected resource strings

Foundation 42 projected `set_InactiveSleepTime` and `set_TargetElapsedTime` with
messages **derived from the resource keys rather than read from the resource
values**. That was wrong, and it is corrected here.

```text
key                              value (the retained Game.dll's .resources stream)
InactiveSleepTimeCannotBeZero    "The inactive sleep time must be greater than or equal
                                  to zero.  Specify zero or a positive value."
TargetElaspedCannotBeZero        "The target elapsed time must be greater than zero.
                                  Specify a non-zero positive value."
```

Two things worth keeping.

`TargetElasped` is Microsoft's own typo in the key name, not a transcription
error, and it is left as it is.

And the correction removes a piece of reasoning that only existed because the
message was wrong. Foundation 42 argued that `InactiveSleepTime` accepts zero
"even though the message says it cannot" — but only the KEY says that. The real
string says "greater than or equal to zero", which is exactly what the IL's
`op_LessThan` admits. The value and the IL agree; it was the invented message
that disagreed with both.

The double spaces before each second sentence are the reference's own and are
preserved, as they are in the service container's messages, which come from the
same stream.
