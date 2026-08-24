# Initial scaffold public-surface audit

Baseline: `1adec27d421745fb176e6cd2cf5f849b3226f891` on `develop`.
“Member” rows include every exported field and method of the initial exported
types. Package documentation declarations are not Go symbols.

| GO_PACKAGE | GO_SYMBOL | XNA_IDENTITY | XNA_KIND | CURRENT_STATUS | REAL_BEHAVIOR | MISSING_BEHAVIOR | MAPPING_JUSTIFIED | KEEP | CHANGE | REMOVE |
|---|---|---|---|---|---|---|---|---|---|---|
| framework | `Vector2` | `Microsoft.Xna.Framework.Vector2` | struct | partial | value struct | 72+ mapped identities and exact behavior | yes | yes | complete it later | no |
| framework | `Vector2.X` | `Vector2.X` | field | real | stores `float32` | none structurally | yes | yes | no | no |
| framework | `Vector2.Y` | `Vector2.Y` | field | real | stores `float32` | none structurally | yes | yes | no | no |
| framework | `NewVector2(float32,float32)` | `Vector2..ctor(float,float)` | constructor | real but misnamed under overload rule | constructs two components | scalar constructor and overload identity | no | no | rename deterministically | yes |
| framework | `Vector2.Add(Vector2)` | no exact member | invented instance method | real arithmetic, wrong identity | component addition | XNA `Add` is static and overloaded; operator is distinct | no | no | replace with mapped static/operator identities | yes |
| framework | `Vector2.LengthSquared()` | `Vector2.LengthSquared()` | method | real | `X*X + Y*Y` in `float32` | edge-case corpus | yes | yes | qualify behavior | no |
| framework | `Color` | `Microsoft.Xna.Framework.Color` | struct | partial | four-byte value | constructors, properties, packed value, named colors, operations | yes | yes | make property distinction explicit | no |
| framework | `Color.R` | `Color.R` | property represented as field | wrong kind | stores red channel | getter/setter property mapping | no | no | replace with methods | yes |
| framework | `Color.G` | `Color.G` | property represented as field | wrong kind | stores green channel | getter/setter property mapping | no | no | replace with methods | yes |
| framework | `Color.B` | `Color.B` | property represented as field | wrong kind | stores blue channel | getter/setter property mapping | no | no | replace with methods | yes |
| framework | `Color.A` | `Color.A` | property represented as field | wrong kind | stores alpha channel | getter/setter property mapping | no | no | replace with methods | yes |
| framework | `NewColor(uint8,uint8,uint8,uint8)` | no XNA constructor signature | constructor helper | clamps nothing and packs bytes | XNA constructors use `int32`, `float32`, `Vector3`, or `Vector4` | no | no | replace with exact overload names | yes |
| framework | `CornflowerBlue` | `Color.CornflowerBlue` | static property | wrong prefix and mutable variable | correct channel tuple | read-only semantics and type prefix | no | no | `ColorCornflowerBlue()` | yes |
| framework | `White` | `Color.White` | static property | wrong prefix and mutable variable | correct channel tuple | read-only semantics and type prefix | no | no | `ColorWhite()` | yes |
| framework | `GameTime` | `Microsoft.Xna.Framework.GameTime` | class adapted as value | wrong shape | callback-sized struct | exact `TimeSpan` values and constructors/getters | partially | yes | replace fields with tick-exact accessors | no |
| framework | `GameTime.TotalSeconds` | `GameTime.TotalGameTime` | property represented as field | wrong name/type/kind | stores seconds as `float64` | exact signed 100ns ticks | no | no | `TotalGameTime() TimeSpan` | yes |
| framework | `GameTime.ElapsedSeconds` | `GameTime.ElapsedGameTime` | property represented as field | wrong name/type/kind | stores seconds as `float64` | exact signed 100ns ticks | no | no | `ElapsedGameTime() TimeSpan` | yes |
| framework | `GameTime.IsRunningSlowly` | `GameTime.IsRunningSlowly` | property represented as field | wrong kind | stores native-independent bool | getter semantics and native data | no | no | `IsRunningSlowly() bool` | yes |
| framework | `Game` | `Microsoft.Xna.Framework.Game` | class mapped as interface | honest but incomplete adaptation | declares five callbacks | XNA host identity, Exit/state, native ownership | no | no | struct host plus `GameCallbacks` adapter | yes |
| framework | `Game.Initialize()` | protected virtual override | interface method | shell | consumer callback | explicit host and native dispatch | no | no | move to `GameCallbacks.Initialize(*Game)` | yes |
| framework | `Game.LoadContent()` | protected virtual override | interface method | shell | consumer callback | explicit host and native dispatch | no | no | move to `GameCallbacks.LoadContent(*Game)` | yes |
| framework | `Game.Update(GameTime)` | protected virtual override | interface method | shell | consumer callback | exact timing and host | no | no | move to `GameCallbacks.Update(*Game,GameTime)` | yes |
| framework | `Game.Draw(GameTime)` | protected virtual override | interface method | shell | consumer callback | exact timing and host | no | no | move to `GameCallbacks.Draw(*Game,GameTime)` | yes |
| framework | `Game.UnloadContent()` | protected virtual override | interface method | shell | consumer callback | explicit host and native dispatch | no | no | move to `GameCallbacks.UnloadContent(*Game)` | yes |
| framework | `Run(Game)` | `Game.Run()` | package adaptation | honestly unavailable | nil check then `ErrNativeUnavailable` | real CNA create/run/exit/destroy lifecycle | no | no | replace with `(*Game).Run()` | yes |

The initial `internal/interop.ErrNativeUnavailable` is not part of an XNA
package. It accurately reported the scaffold state, but the admitted loader now
uses structured load/version/symbol/result errors instead of a permanent
sentinel.
