# 2026-11-21: a delta's markup, offered to its Screen

Branch `delta-offer`, off `doc-offer`. Not pushed. Internal: no API
change. Step 3 of making new blocks in cells cheaper, after the
document offer (2026-11-21-document-offer.md).

## Why

The document offer spared a new block's second parse. Every delta with
markup (an inner, a replace, a before, an append) was still parsed by
the Screen from the bytes surf had just rendered from its own nodes.

## What changed

- **`rt.Host.OfferDelta(name, frags)`.** Each `rt.Fragment` is an op's
  markup and the nodes it renders, in the order of the ops that carry
  markup. surf's differ collects them as it renders (`render`, `inner`)
  only where the surface has a host. The surface offers them as it
  writes the delta, lent for the call, as the document's nodes are.
- **The copy, in cells, inside the call.** Each fragment gets a copy of
  its own (`copyNodes`), its list capped at its length. One copy for the
  whole delta shared arrays and a string buffer among its fragments, so
  a row a log keeps kept every other fragment of its step, such as a
  status written again. A verifier's probe showed it: after 400 steps a
  Screen with offers held 11.2 MB where a parsing one held 420 KB. With
  a copy a fragment it holds what a parse does.
- **Taken by bytes.** The next `a=delta` for the surface takes the
  offer. Each op with markup (`fragment`) takes the next fragment's copy
  where its markup is that fragment's, byte for byte. From the first op
  of other bytes, the rest of the delta is parsed.
  - Any `a=delta` ends the offer, another surface's too, and so do
    `a=doc`, `a=del` and the next offer.
  - The delta stays on the wire whole, so no channel changes (P-3
    doesn't apply).

## Proof

- **The oracle** (`checked.OfferDelta`) checks each fragment's copy.
  It must be the surface's nodes, sharing nothing, with no links, and
  with no room for the next fragment's. The markup's parse under the
  nodes' parent must be those nodes too, or else the difference is
  noted. None was.
- **TestDeltaOffersAreTheTree.** 60 random documents each take 30
  steps: another document's markup, a part added, or random texts and
  attributes. That makes every op the differ writes. The Screen that
  takes offers is held to one that parses, cell for cell. 6,147
  fragments were offered, and the Screen's trees took 17,545 of their
  nodes.
- **TestOfferIsTheTree** (the stories' walks) counts the fragments
  taken as well.
- **TestDeltaOfferEdges** covers:
  - fragments taken in order past ops without markup;
  - the same delta twice;
  - other bytes, after which a fragment's bytes are parsed too;
  - another surface's delta and another surface's offer;
  - a document between, and `a=del` and the delete of every surface;
  - a compressed delta;
  - nodes changed after the call;
  - each fragment's nodes capped.
- **TestDeltaOffersRetainWhatAParseDoes.** A log appends a row each
  step beside a 60-span status written again. Over 300 steps, the Screen
  with offers keeps no more than 1.5 times what a parsing one keeps,
  plus 64 KB: 359 KB against 358 KB. One copy for the delta keeps
  8,433 KB.
- **TestDeltaOffersThroughFill.** Compiled rows in a column and in a
  tbody change at random through `fill` (fragments parsed in place by
  `parseIn`, then `restop`). They're held to the oracle, with any
  divergence an error, and to a parsing Screen node for node and cell
  for cell. 780 fragments were offered, all taken.
- A fragment's divergence fails TestOfferIsTheTree and
  TestDeltaOffersAreTheTree. A document's is logged, as before.
- **FuzzHandoff** takes two renders now: a document, then the delta to
  the next.
- **TestDeltaOfferSparesTheParse.** Rows that come and go take 395 → 253
  allocations and 63.8 → 52.1 KB a step.
- Each guarantee was sabotaged and caught:
  - no markup check;
  - no stop at other bytes;
  - no name check;
  - an offer kept past another surface's delta, a document or `a=del`;
  - room left for the next fragment;
  - no copy;
  - one copy for the whole delta (the retention test).

## Measured

Main 44b002b, doc-offer and this branch, fastest of four interleaved
runs, under the build cap, at a load of 11 to 13 (so the times are
rough). The bench step is 12 rows written again, on a Screen.

| bench | main | doc-offer | delta-offer |
| --- | --- | --- | --- |
| step on a Screen | 254,750 B, 1,274 allocs | 254,577 B, 1,274 | 234,162 B, 1,170 |
| new pill | 75,182 B, 672 allocs | 66,996 B, 408 | 66,996 B, 408 |

hello.wasm under TinyGo: 2,503,141 to 2,504,013 B. That's the fragments'
collection, which a browser build links but never runs, since a page's
surface has no in-process host. Leaving it out would need a build tag
on the differ, which isn't worth 872 B.
