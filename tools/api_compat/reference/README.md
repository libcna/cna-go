# Pinned XNA reference contract

`xna40-windows-runtime-contract.json` is an unchanged retained measurement of
the Microsoft XNA Framework 4.0 Windows runtime public metadata. It is input to
CNA-Go's language mapper; it is not generated from CNA, CNA-Go, MonoGame, FNA,
or another binding's public surface.

- Profile: `XNA 4.0 Windows runtime`
- Types: 257
- Members: 2,964
- Contract schema: 2
- Contract SHA-256: `7207908eb7926cc90a156d0370c907add4dda465421cea1cbec51afba2f97fdc`
- Retained source: `openeggbert/cna-python`
- Source path: `tools/api_compat/reference/xna40-windows-runtime-contract.json`
- Introducing source commit: `712ab83a3a39681838a765849b6659e31c8bfbde`

The retained profile records the seven assembly identities and SHA-256 values
used to produce the contract:

| Assembly | SHA-256 |
|---|---|
| Microsoft.Xna.Framework.dll | `38e7093f52d7474bbc6256906519781a1210d7da50a1c667b52716fcf49ca130` |
| Microsoft.Xna.Framework.Game.dll | `b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0` |
| Microsoft.Xna.Framework.Graphics.dll | `560080fc39021c611ca9d076dcebed312faf6d7d1413c2dc523683ea635e9f55` |
| Microsoft.Xna.Framework.Storage.dll | `798f678e9ae3d9afc3bed66c30123bc9634fb923b6d200188344b618e608cbb8` |
| Microsoft.Xna.Framework.Video.dll | `17538b1ca9d48a993e2cd88c96b436df08e7abb4aec5d4758eb21feb580d6e06` |
| Microsoft.Xna.Framework.Input.Touch.dll | `b0585224c18022c3661057ae79544644c10f33f1dc529678364f3d6b25151c25` |
| Microsoft.Xna.Framework.Xact.dll | `a14d5364dca7cf49fb90639e87ba04d52b59a700dc9198efa5707ce8eae28f0a` |

The proprietary assemblies are not redistributed. To regenerate, a legally
supplied local reference pack must be measured with the metadata extractor in
the retained source repository, then the counts, assembly hashes, contract
hash, and representative signatures must be independently compared before
this file is replaced.
